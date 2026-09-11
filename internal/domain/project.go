package domain

// Project represents a project that groups related work items.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"` // "active", "completed", "archived"
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Project status constants
const (
	ProjectStatusActive    = "active"
	ProjectStatusCompleted = "completed"
	ProjectStatusArchived  = "archived"
)
