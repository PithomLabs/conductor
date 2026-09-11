package agentstub

import (
	"context"
	"fmt"
	"time"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/mcp"
	"github.com/PithomLabs/conductor/internal/store"
)

// Stub is a simulated agent that completes tasks through MCP.
// It interacts only through the MCP surface, not repositories directly.
type Stub struct {
	mcpServer *mcp.Server
	agentID   string
	governance governance.GovernanceReader
}

// NewStub creates a new agent stub
func NewStub(db *store.DB, gov governance.GovernanceReader, agentID string) *Stub {
	return &Stub{
		mcpServer: mcp.NewServer(db, gov, agentID),
		agentID:   agentID,
		governance: gov,
	}
}

// Run executes the full task lifecycle through MCP
func (s *Stub) Run(ctx context.Context, projectID string) error {
	fmt.Printf("[%s] Starting task lifecycle for project %s\n", s.agentID, projectID)

	// 1. Discover task
	task, err := s.discoverTask(ctx, projectID)
	if err != nil {
		return fmt.Errorf("discover task: %w", err)
	}
	fmt.Printf("[%s] Discovered task: %s\n", s.agentID, task.ID)

	// 2. Claim task
	err = s.claimTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("claim task: %w", err)
	}
	fmt.Printf("[%s] Claimed task: %s\n", s.agentID, task.ID)

	// 3. Perform simulated work
	err = s.performWork(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("perform work: %w", err)
	}
	fmt.Printf("[%s] Performed work on task: %s\n", s.agentID, task.ID)

	// 4. Record activity
	err = s.recordActivity(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("record activity: %w", err)
	}
	fmt.Printf("[%s] Recorded activity for task: %s\n", s.agentID, task.ID)

	// 5. Submit task
	err = s.submitTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("submit task: %w", err)
	}
	fmt.Printf("[%s] Submitted task: %s\n", s.agentID, task.ID)

	return nil
}

// discoverTask finds the next unblocked task through MCP
func (s *Stub) discoverTask(ctx context.Context, projectID string) (*domain.Task, error) {
	resp := s.mcpServer.CallTool("conductor_next_task", map[string]interface{}{
		"project_id": projectID,
	})
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	task, ok := resp.Result.(*domain.Task)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	return task, nil
}

// claimTask claims the task through MCP
func (s *Stub) claimTask(ctx context.Context, taskID string) error {
	resp := s.mcpServer.CallTool("conductor_claim_task", map[string]interface{}{
		"task_id": taskID,
	})
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// performWork simulates work (sleep to simulate time passing)
func (s *Stub) performWork(ctx context.Context, taskID string) error {
	// Simulate work by sleeping
	time.Sleep(100 * time.Millisecond)
	return nil
}

// recordActivity records work activity through MCP
func (s *Stub) recordActivity(ctx context.Context, taskID string) error {
	resp := s.mcpServer.CallTool("conductor_post_activity", map[string]interface{}{
		"task_id": taskID,
		"action":  "work.completed",
		"details": fmt.Sprintf("Agent %s completed simulated work", s.agentID),
	})
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// submitTask submits the task for review through MCP
func (s *Stub) submitTask(ctx context.Context, taskID string) error {
	resp := s.mcpServer.CallTool("conductor_submit_task", map[string]interface{}{
		"task_id": taskID,
	})
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// RunWithGovernance exercises the full lifecycle including governance observation
func (s *Stub) RunWithGovernance(ctx context.Context, projectID string) error {
	fmt.Printf("[%s] Starting task lifecycle with governance for project %s\n", s.agentID, projectID)

	// 1. Discover task
	task, err := s.discoverTask(ctx, projectID)
	if err != nil {
		return fmt.Errorf("discover task: %w", err)
	}
	fmt.Printf("[%s] Discovered task: %s\n", s.agentID, task.ID)

	// 2. Query governance if task has governance_ref
	if task.GovernanceRef != nil && *task.GovernanceRef != "" {
		fmt.Printf("[%s] Task has governance_ref, querying governance\n", s.agentID)
		// Query governance through MCP
		resp := s.mcpServer.CallTool("conductor_get_governance", map[string]interface{}{
			"task_id": task.ID,
		})
		if resp.Error != "" {
			fmt.Printf("[%s] Governance query failed: %s\n", s.agentID, resp.Error)
		} else {
			fmt.Printf("[%s] Governance state: %v\n", s.agentID, resp.Result)
		}
	}

	// 3. Claim task
	err = s.claimTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("claim task: %w", err)
	}
	fmt.Printf("[%s] Claimed task: %s\n", s.agentID, task.ID)

	// 4. Perform simulated work
	err = s.performWork(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("perform work: %w", err)
	}
	fmt.Printf("[%s] Performed work on task: %s\n", s.agentID, task.ID)

	// 5. Record activity
	err = s.recordActivity(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("record activity: %w", err)
	}
	fmt.Printf("[%s] Recorded activity for task: %s\n", s.agentID, task.ID)

	// 6. Submit task
	err = s.submitTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("submit task: %w", err)
	}
	fmt.Printf("[%s] Submitted task: %s\n", s.agentID, task.ID)

	return nil
}
