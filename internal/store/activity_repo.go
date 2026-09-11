package store

import (
	"context"
	"fmt"

	"github.com/PithomLabs/conductor/internal/domain"
)

// ActivityRepository handles activity persistence (append-only)
type ActivityRepository struct {
	db *DB
}

// NewActivityRepository creates a new ActivityRepository
func NewActivityRepository(db *DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Append appends a new activity event (append-only, never updates or deletes)
func (r *ActivityRepository) Append(ctx context.Context, activity *domain.Activity) error {
	if activity.ID == "" {
		return fmt.Errorf("activity ID is required")
	}
	if activity.TaskID == "" {
		return fmt.Errorf("activity task_id is required")
	}
	if activity.ActorType == "" {
		return fmt.Errorf("activity actor_type is required")
	}
	if activity.ActorID == "" {
		return fmt.Errorf("activity actor_id is required")
	}
	if activity.Action == "" {
		return fmt.Errorf("activity action is required")
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO conductor_activity (id, task_id, actor_type, actor_id, action, details, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
		activity.ID, activity.TaskID, activity.ActorType, activity.ActorID, activity.Action, activity.Details)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}

	return nil
}

// ListByTask returns all activities for a task (ordered by creation time descending)
func (r *ActivityRepository) ListByTask(ctx context.Context, taskID string) ([]*domain.Activity, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, task_id, actor_type, actor_id, action, details, created_at
		 FROM conductor_activity WHERE task_id = ? ORDER BY created_at DESC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	defer rows.Close()

	var activities []*domain.Activity
	for rows.Next() {
		activity := &domain.Activity{}
		if err := rows.Scan(
			&activity.ID, &activity.TaskID, &activity.ActorType, &activity.ActorID,
			&activity.Action, &activity.Details, &activity.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		activities = append(activities, activity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activities: %w", err)
	}

	return activities, nil
}

// ListByProject returns all activities for a project (ordered by creation time descending)
func (r *ActivityRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Activity, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT a.id, a.task_id, a.actor_type, a.actor_id, a.action, a.details, a.created_at
		 FROM conductor_activity a
		 JOIN conductor_task t ON a.task_id = t.id
		 WHERE t.project_id = ? ORDER BY a.created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	defer rows.Close()

	var activities []*domain.Activity
	for rows.Next() {
		activity := &domain.Activity{}
		if err := rows.Scan(
			&activity.ID, &activity.TaskID, &activity.ActorType, &activity.ActorID,
			&activity.Action, &activity.Details, &activity.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		activities = append(activities, activity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activities: %w", err)
	}

	return activities, nil
}
