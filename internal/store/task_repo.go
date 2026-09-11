package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/google/uuid"
)

// TaskRepository handles task persistence with lifecycle enforcement
type TaskRepository struct {
	db *DB
}

// NewTaskRepository creates a new TaskRepository
func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create inserts a new task.
// New tasks must enter proposed/unassigned state.
// Caller-supplied status or current_agent are rejected.
func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if task.ProjectID == "" {
		return fmt.Errorf("task project_id is required")
	}
	if task.Title == "" {
		return fmt.Errorf("task title is required")
	}

	// Reject contradictory caller input — lifecycle fields must not be set at creation
	if task.Status != "" && task.Status != domain.TaskStatusProposed {
		return ErrLifecycleBypass
	}
	if task.CurrentAgent != nil {
		return ErrLifecycleBypass
	}

	// Enforce new task invariants
	task.Status = domain.TaskStatusProposed
	task.CurrentAgent = nil

	now := time.Now().UTC().Format(time.RFC3339)
	if task.Priority == "" {
		task.Priority = domain.TaskPriorityMedium
	}
	task.CreatedAt = now
	task.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO conductor_task (id, project_id, title, description, status, priority, current_agent, governance_ref, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.ProjectID, task.Title, task.Description, task.Status, task.Priority,
		task.CurrentAgent, task.GovernanceRef, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

// GetByID retrieves a task by ID
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	task := &domain.Task{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, title, description, status, priority, current_agent, governance_ref, created_at, updated_at
		 FROM conductor_task WHERE id = ?`, id).Scan(
		&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status,
		&task.Priority, &task.CurrentAgent, &task.GovernanceRef,
		&task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return task, nil
}

// Update updates ordinary task fields (NOT status, current_agent, or governance_ref)
func (r *TaskRepository) Update(ctx context.Context, taskID string, fields TaskUpdateFields) error {
	if taskID == "" {
		return fmt.Errorf("task ID is required")
	}

	setClauses := []string{}
	args := []interface{}{}

	if fields.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *fields.Title)
	}
	if fields.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *fields.Description)
	}
	if fields.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *fields.Priority)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, taskID)

	query := fmt.Sprintf("UPDATE conductor_task SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// TaskUpdateFields defines which fields can be updated via generic PATCH.
// GovernanceRef is excluded — it is write-once at creation.
// Status and CurrentAgent are excluded — they change only through lifecycle operations.
type TaskUpdateFields struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Priority    *string `json:"priority,omitempty"`
}

// validateTransition checks if a transition from one status to another is allowed
func validateTransition(fromStatus, toStatus string) bool {
	allowed := map[string][]string{
		domain.TaskStatusProposed: {domain.TaskStatusActive, domain.TaskStatusBlocked, domain.TaskStatusCancelled},
		domain.TaskStatusActive:   {domain.TaskStatusReview, domain.TaskStatusBlocked, domain.TaskStatusCancelled, domain.TaskStatusProposed},
		domain.TaskStatusReview:   {domain.TaskStatusAccepted, domain.TaskStatusActive, domain.TaskStatusCancelled},
		domain.TaskStatusBlocked:  {domain.TaskStatusActive, domain.TaskStatusCancelled},
	}
	transitions, ok := allowed[fromStatus]
	if !ok {
		return false
	}
	for _, valid := range transitions {
		if valid == toStatus {
			return true
		}
	}
	return false
}

// Transition performs a lifecycle transition atomically with activity event
func (r *TaskRepository) Transition(ctx context.Context, taskID string, fromStatus string, toStatus string, actorType string, actorID string, action string, details string) error {
	if !validateTransition(fromStatus, toStatus) {
		return ErrInvalidTransition
	}

	return r.db.Transaction(func(tx *sql.Tx) error {
		var currentStatus string
		err := tx.QueryRowContext(ctx, "SELECT status FROM conductor_task WHERE id = ?", taskID).Scan(&currentStatus)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get task status: %w", err)
		}
		if currentStatus != fromStatus {
			return ErrInvalidTransition
		}

		_, err = tx.ExecContext(ctx,
			"UPDATE conductor_task SET status = ?, updated_at = datetime('now') WHERE id = ?",
			toStatus, taskID)
		if err != nil {
			return fmt.Errorf("update task status: %w", err)
		}

		activityID := generateUUID()
		_, err = tx.ExecContext(ctx,
			`INSERT INTO conductor_activity (id, task_id, actor_type, actor_id, action, details, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
			activityID, taskID, actorType, actorID, action, details)
		if err != nil {
			return fmt.Errorf("insert activity: %w", err)
		}

		return nil
	})
}

// TransitionWithAgent performs a lifecycle transition that also sets current_agent
func (r *TaskRepository) TransitionWithAgent(ctx context.Context, taskID string, fromStatus string, toStatus string, agentID string, action string, details string) error {
	if !validateTransition(fromStatus, toStatus) {
		return ErrInvalidTransition
	}

	return r.db.Transaction(func(tx *sql.Tx) error {
		var currentStatus string
		err := tx.QueryRowContext(ctx, "SELECT status FROM conductor_task WHERE id = ?", taskID).Scan(&currentStatus)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get task status: %w", err)
		}
		if currentStatus != fromStatus {
			return ErrInvalidTransition
		}

		_, err = tx.ExecContext(ctx,
			"UPDATE conductor_task SET status = ?, current_agent = ?, updated_at = datetime('now') WHERE id = ?",
			toStatus, agentID, taskID)
		if err != nil {
			return fmt.Errorf("update task status: %w", err)
		}

		activityID := generateUUID()
		_, err = tx.ExecContext(ctx,
			`INSERT INTO conductor_activity (id, task_id, actor_type, actor_id, action, details, created_at)
			 VALUES (?, ?, 'agent', ?, ?, ?, datetime('now'))`,
			activityID, taskID, agentID, action, details)
		if err != nil {
			return fmt.Errorf("insert activity: %w", err)
		}

		return nil
	})
}

// Claim atomically claims a task using compare-and-swap
func (r *TaskRepository) Claim(ctx context.Context, taskID string, agentID string) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx,
			`UPDATE conductor_task
			 SET current_agent = ?, status = 'active', updated_at = datetime('now')
			 WHERE id = ? AND status = 'proposed' AND current_agent IS NULL`,
			agentID, taskID)
		if err != nil {
			return fmt.Errorf("claim task: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("get rows affected: %w", err)
		}
		if rowsAffected == 0 {
			return ErrClaimFailed
		}

		activityID := generateUUID()
		_, err = tx.ExecContext(ctx,
			`INSERT INTO conductor_activity (id, task_id, actor_type, actor_id, action, details, created_at)
			 VALUES (?, ?, 'agent', ?, 'task.claimed', '{}', datetime('now'))`,
			activityID, taskID, agentID)
		if err != nil {
			return fmt.Errorf("insert activity: %w", err)
		}

		return nil
	})
}


// Release releases a task back to proposed state (active -> proposed).
// Only the assigned agent may release. Enforced atomically via SQL WHERE clause.
func (r *TaskRepository) Release(ctx context.Context, taskID string, agentID string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE conductor_task
		 SET current_agent = NULL, status = 'proposed', updated_at = datetime('now')
		 WHERE id = ? AND current_agent = ? AND status = 'active'`,
		taskID, agentID)
	if err != nil {
		return fmt.Errorf("release task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrReleaseFailed
	}

	activityID := generateUUID()
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO conductor_activity (id, task_id, actor_type, actor_id, action, details, created_at)
		 VALUES (?, ?, 'agent', ?, 'task.released', '{}', datetime('now'))`,
		activityID, taskID, agentID)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}

	return nil
}

// ListByProject returns all tasks for a project
func (r *TaskRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, title, description, status, priority, current_agent, governance_ref, created_at, updated_at
		 FROM conductor_task WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task := &domain.Task{}
		if err := rows.Scan(
			&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status,
			&task.Priority, &task.CurrentAgent, &task.GovernanceRef,
			&task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

// ListByStatus returns all tasks with a given status
func (r *TaskRepository) ListByStatus(ctx context.Context, status string) ([]*domain.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, title, description, status, priority, current_agent, governance_ref, created_at, updated_at
		 FROM conductor_task WHERE status = ? ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task := &domain.Task{}
		if err := rows.Scan(
			&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status,
			&task.Priority, &task.CurrentAgent, &task.GovernanceRef,
			&task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

// generateUUID generates a collision-resistant unique identifier
func generateUUID() string {
	return uuid.New().String()
}
