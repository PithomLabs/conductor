package domain

// Task represents a unit of work that agents can claim and execute.
type Task struct {
	ID            string  `json:"id"`
	ProjectID     string  `json:"project_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description,omitempty"`
	Status        string  `json:"status"` // "proposed", "active", "review", "accepted", "blocked", "cancelled"
	Priority      string  `json:"priority"` // "low", "medium", "high", "critical"
	CurrentAgent  *string `json:"current_agent,omitempty"`
	GovernanceRef *string `json:"governance_ref,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// Task status constants
const (
	TaskStatusProposed  = "proposed"
	TaskStatusActive    = "active"
	TaskStatusReview    = "review"
	TaskStatusAccepted  = "accepted"
	TaskStatusBlocked   = "blocked"
	TaskStatusCancelled = "cancelled"
)

// Task priority constants
const (
	TaskPriorityLow      = "low"
	TaskPriorityMedium   = "medium"
	TaskPriorityHigh     = "high"
	TaskPriorityCritical = "critical"
)

// IsTerminal returns true if the task status is terminal.
func (t *Task) IsTerminal() bool {
	return t.Status == TaskStatusAccepted || t.Status == TaskStatusCancelled
}

// CanTransitionTo checks if a transition from current status to target is allowed.
func (t *Task) CanTransitionTo(target string) bool {
	allowed := map[string][]string{
		TaskStatusProposed: {TaskStatusActive, TaskStatusBlocked, TaskStatusCancelled},
		TaskStatusActive:   {TaskStatusReview, TaskStatusBlocked, TaskStatusCancelled, TaskStatusProposed},
		TaskStatusReview:   {TaskStatusAccepted, TaskStatusActive, TaskStatusCancelled},
		TaskStatusBlocked:  {TaskStatusActive, TaskStatusCancelled},
	}
	transitions, ok := allowed[t.Status]
	if !ok {
		return false
	}
	for _, valid := range transitions {
		if valid == target {
			return true
		}
	}
	return false
}
