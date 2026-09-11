package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/PithomLabs/conductor/internal/domain"
)

// ProjectRepository handles project persistence
type ProjectRepository struct {
	db *DB
}

// NewProjectRepository creates a new ProjectRepository
func NewProjectRepository(db *DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create inserts a new project
func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	if project.ID == "" {
		return fmt.Errorf("project ID is required")
	}
	if project.Name == "" {
		return fmt.Errorf("project name is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if project.Status == "" {
		project.Status = domain.ProjectStatusActive
	}
	project.CreatedAt = now
	project.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO conductor_project (id, name, description, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		project.ID, project.Name, project.Description, project.Status, project.CreatedAt, project.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}

	return nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	project := &domain.Project{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, status, created_at, updated_at
		 FROM conductor_project WHERE id = ?`, id).Scan(
		&project.ID, &project.Name, &project.Description, &project.Status,
		&project.CreatedAt, &project.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return project, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	if project.ID == "" {
		return fmt.Errorf("project ID is required")
	}

	project.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	result, err := r.db.ExecContext(ctx,
		`UPDATE conductor_project
		 SET name = ?, description = ?, status = ?, updated_at = ?
		 WHERE id = ?`,
		project.Name, project.Description, project.Status, project.UpdatedAt, project.ID)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
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

// Delete deletes a project
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM conductor_project WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
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

// List returns all projects
func (r *ProjectRepository) List(ctx context.Context) ([]*domain.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description, status, created_at, updated_at
		 FROM conductor_project ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		project := &domain.Project{}
		if err := rows.Scan(
			&project.ID, &project.Name, &project.Description, &project.Status,
			&project.CreatedAt, &project.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}

	return projects, nil
}
