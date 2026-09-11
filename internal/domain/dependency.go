package domain

// Dependency represents a blocking relationship between tasks.
type Dependency struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`      // The task that is blocked
	BlockedByID string `json:"blocked_by_id"` // The task that blocks
	CreatedAt   string `json:"created_at"`
}
