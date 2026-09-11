package agentstub

import (
	"context"
	"fmt"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/mcp"
	"github.com/PithomLabs/conductor/internal/store"
)

// Reviewer simulates a human reviewer
type Reviewer struct {
	mcpServer *mcp.Server
	agentID   string
}

// NewReviewer creates a new reviewer
func NewReviewer(db *store.DB, gov governance.GovernanceReader, agentID string) *Reviewer {
	return &Reviewer{
		mcpServer: mcp.NewServer(db, gov, agentID),
		agentID:   agentID,
	}
}

// ReviewTask reviews and accepts/rejects a task through MCP
func (r *Reviewer) ReviewTask(ctx context.Context, taskID string, accept bool) error {
	fmt.Printf("[%s] Reviewing task: %s (accept: %v)\n", r.agentID, taskID, accept)

	if accept {
		resp := r.mcpServer.CallTool("conductor_accept_task", map[string]interface{}{
			"task_id": taskID,
		})
		if resp.Error != "" {
			return fmt.Errorf("%s", resp.Error)
		}
		fmt.Printf("[%s] Accepted task: %s\n", r.agentID, taskID)
	} else {
		resp := r.mcpServer.CallTool("conductor_reject_task", map[string]interface{}{
			"task_id": taskID,
		})
		if resp.Error != "" {
			return fmt.Errorf("%s", resp.Error)
		}
		fmt.Printf("[%s] Rejected task: %s\n", r.agentID, taskID)
	}

	return nil
}

// ReviewSubmittedTask finds and reviews a submitted task
func (r *Reviewer) ReviewSubmittedTask(ctx context.Context, projectID string, accept bool) error {
	// Get tasks for project
	resp := r.mcpServer.CallTool("conductor_list_tasks", map[string]interface{}{
		"project_id": projectID,
	})
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}

	tasks, ok := resp.Result.([]*domain.Task)
	if !ok {
		return fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	// Find first task in review status
	for _, task := range tasks {
		if task.Status == domain.TaskStatusReview {
			return r.ReviewTask(ctx, task.ID, accept)
		}
	}

	return fmt.Errorf("no submitted tasks found")
}
