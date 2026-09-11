package domain

// Activity represents an immutable event log entry for task mutations and agent actions.
type Activity struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	ActorType string `json:"actor_type"` // "human", "agent", "system"
	ActorID   string `json:"actor_id"`
	Action    string `json:"action"` // Event type (e.g., "task.claimed", "task.submitted")
	Details   string `json:"details,omitempty"` // JSON metadata (non-authoritative)
	CreatedAt string `json:"created_at"`
}

// Activity actor type constants
const (
	ActorTypeHuman  = "human"
	ActorTypeAgent  = "agent"
	ActorTypeSystem = "system"
)

// Activity action constants
const (
	ActionTaskClaimed   = "task.claimed"
	ActionTaskSubmitted = "task.submitted"
	ActionTaskAccepted  = "task.accepted"
	ActionTaskRejected  = "task.rejected"
	ActionTaskCancelled = "task.cancelled"
	ActionTaskBlocked   = "task.blocked"
	ActionTaskUnblocked = "task.unblocked"
	ActionTaskUpdated   = "task.updated"
	ActionTaskReleased = "task.released"
)
