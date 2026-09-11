package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/PithomLabs/conductor/internal/domain"
)

// DependencyRepository handles dependency persistence with same-project validation
type DependencyRepository struct {
	db *DB
}

// NewDependencyRepository creates a new DependencyRepository
func NewDependencyRepository(db *DB) *DependencyRepository {
	return &DependencyRepository{db: db}
}

// Create inserts a new dependency (with same-project validation)
func (r *DependencyRepository) Create(ctx context.Context, dep *domain.Dependency) error {
	if dep.ID == "" {
		return fmt.Errorf("dependency ID is required")
	}
	if dep.TaskID == "" {
		return fmt.Errorf("dependency task_id is required")
	}
	if dep.BlockedByID == "" {
		return fmt.Errorf("dependency blocked_by_id is required")
	}
	if dep.TaskID == dep.BlockedByID {
		return fmt.Errorf("task cannot depend on itself")
	}

	// Validate both tasks exist and belong to the same project
	var taskProjectID, blockedByProjectID string
	err := r.db.QueryRowContext(ctx,
		"SELECT project_id FROM conductor_task WHERE id = ?", dep.TaskID).Scan(&taskProjectID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get task project: %w", err)
	}

	err = r.db.QueryRowContext(ctx,
		"SELECT project_id FROM conductor_task WHERE id = ?", dep.BlockedByID).Scan(&blockedByProjectID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get blocked_by task project: %w", err)
	}

	if taskProjectID != blockedByProjectID {
		return ErrSameProject
	}

	dep.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO conductor_dependency (id, task_id, blocked_by_id, created_at)
		 VALUES (?, ?, ?, ?)`,
		dep.ID, dep.TaskID, dep.BlockedByID, dep.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert dependency: %w", err)
	}

	return nil
}

// Delete deletes a dependency
func (r *DependencyRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM conductor_dependency WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete dependency: %w", err)
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

// ListByTask returns all dependencies where task_id matches
func (r *DependencyRepository) ListByTask(ctx context.Context, taskID string) ([]*domain.Dependency, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, task_id, blocked_by_id, created_at
		 FROM conductor_dependency WHERE task_id = ? ORDER BY created_at`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list dependencies: %w", err)
	}
	defer rows.Close()

	var deps []*domain.Dependency
	for rows.Next() {
		dep := &domain.Dependency{}
		if err := rows.Scan(&dep.ID, &dep.TaskID, &dep.BlockedByID, &dep.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan dependency: %w", err)
		}
		deps = append(deps, dep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dependencies: %w", err)
	}

	return deps, nil
}

// ListByBlockedBy returns all dependencies where blocked_by_id matches
func (r *DependencyRepository) ListByBlockedBy(ctx context.Context, blockedByID string) ([]*domain.Dependency, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, task_id, blocked_by_id, created_at
		 FROM conductor_dependency WHERE blocked_by_id = ? ORDER BY created_at`, blockedByID)
	if err != nil {
		return nil, fmt.Errorf("list dependencies: %w", err)
	}
	defer rows.Close()

	var deps []*domain.Dependency
	for rows.Next() {
		dep := &domain.Dependency{}
		if err := rows.Scan(&dep.ID, &dep.TaskID, &dep.BlockedByID, &dep.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan dependency: %w", err)
		}
		deps = append(deps, dep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dependencies: %w", err)
	}

	return deps, nil
}

// GetBlockedTasks returns all task IDs that are blocked by a given task
func (r *DependencyRepository) GetBlockedTasks(ctx context.Context, blockingTaskID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT task_id FROM conductor_dependency WHERE blocked_by_id = ?", blockingTaskID)
	if err != nil {
		return nil, fmt.Errorf("get blocked tasks: %w", err)
	}
	defer rows.Close()

	var taskIDs []string
	for rows.Next() {
		var taskID string
		if err := rows.Scan(&taskID); err != nil {
			return nil, fmt.Errorf("scan task_id: %w", err)
		}
		taskIDs = append(taskIDs, taskID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocked tasks: %w", err)
	}

	return taskIDs, nil
}
