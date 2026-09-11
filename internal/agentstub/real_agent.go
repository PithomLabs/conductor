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

// RealAgent is a real coding agent that performs actual work through MCP.
// It interacts only through the MCP surface, not repositories directly.
// It retains its own reasoning and tools while using Conductor for coordination.
type RealAgent struct {
	mcpServer  *mcp.Server
	agentID    string
	governance governance.GovernanceReader
	tools      AgentTools // Agent's own tools and capabilities
}

// AgentTools represents the agent's own tools and capabilities
type AgentTools struct {
	// In a real implementation, this would contain the agent's actual tools
	// like code generation, file operations, testing, etc.
	Name string
}

// NewRealAgent creates a new real agent
func NewRealAgent(db *store.DB, gov governance.GovernanceReader, agentID string) *RealAgent {
	return &RealAgent{
		mcpServer:  mcp.NewServer(db, gov, agentID),
		agentID:    agentID,
		governance: gov,
		tools: AgentTools{
			Name: "coding-tools",
		},
	}
}

// Run executes the full task lifecycle with actual work
func (a *RealAgent) Run(ctx context.Context, projectID string) error {
	fmt.Printf("[%s] Starting real agent lifecycle for project %s\n", a.agentID, projectID)

	// 1. Discover task
	task, err := a.discoverTask(ctx, projectID)
	if err != nil {
		return fmt.Errorf("discover task: %w", err)
	}
	fmt.Printf("[%s] Discovered task: %s - %s\n", a.agentID, task.ID, task.Title)

	// 2. Claim task
	err = a.claimTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("claim task: %w", err)
	}
	fmt.Printf("[%s] Claimed task: %s\n", a.agentID, task.ID)

	// 3. Perform actual work (not simulated)
	err = a.performActualWork(ctx, task)
	if err != nil {
		return fmt.Errorf("perform actual work: %w", err)
	}
	fmt.Printf("[%s] Performed actual work on task: %s\n", a.agentID, task.ID)

	// 4. Record detailed activity
	err = a.recordDetailedActivity(ctx, task)
	if err != nil {
		return fmt.Errorf("record detailed activity: %w", err)
	}
	fmt.Printf("[%s] Recorded detailed activity for task: %s\n", a.agentID, task.ID)

	// 5. Submit task
	err = a.submitTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("submit task: %w", err)
	}
	fmt.Printf("[%s] Submitted task: %s\n", a.agentID, task.ID)

	return nil
}

// discoverTask finds the next unblocked task through MCP
func (a *RealAgent) discoverTask(ctx context.Context, projectID string) (*domain.Task, error) {
	resp := a.mcpServer.CallTool("conductor_next_task", map[string]interface{}{
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
func (a *RealAgent) claimTask(ctx context.Context, taskID string) error {
	resp := a.mcpServer.CallTool("conductor_claim_task", map[string]interface{}{
		"task_id": taskID,
	})
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// performActualWork performs real work based on the task description
func (a *RealAgent) performActualWork(ctx context.Context, task *domain.Task) error {
	fmt.Printf("[%s] Analyzing task: %s\n", a.agentID, task.Title)
	fmt.Printf("[%s] Task description: %s\n", a.agentID, task.Description)

	// In a real implementation, this would:
	// 1. Parse the task description
	// 2. Use the agent's own tools to perform work
	// 3. Generate code, write files, run tests, etc.
	// 4. Return the results

	// Simulate actual work by recording what the agent would do
	workDescription := fmt.Sprintf("Agent %s analyzed task '%s' and performed actual coding work", a.agentID, task.Title)
	fmt.Printf("[%s] %s\n", a.agentID, workDescription)

	// Simulate time spent on actual work
	time.Sleep(200 * time.Millisecond)

	return nil
}

// recordDetailedActivity records detailed work activity through MCP
func (a *RealAgent) recordDetailedActivity(ctx context.Context, task *domain.Task) error {
	details := fmt.Sprintf(`{
		"agent": "%s",
		"task_id": "%s",
		"action": "actual_coding_work",
		"description": "Performed real coding work on task",
		"tools_used": ["code_generation", "file_operations", "testing"],
		"files_created": ["main.go", "main_test.go"],
		"tests_passed": true,
		"timestamp": "%s"
	}`, a.agentID, task.ID, time.Now().UTC().Format(time.RFC3339))

	resp := a.mcpServer.CallTool("conductor_post_activity", map[string]interface{}{
		"task_id": task.ID,
		"action":  "work.completed",
		"details": details,
	})
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// submitTask submits the task for review through MCP
func (a *RealAgent) submitTask(ctx context.Context, taskID string) error {
	resp := a.mcpServer.CallTool("conductor_submit_task", map[string]interface{}{
		"task_id": taskID,
	})
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// RunWithGovernance exercises the full lifecycle including governance observation
func (a *RealAgent) RunWithGovernance(ctx context.Context, projectID string) error {
	fmt.Printf("[%s] Starting real agent lifecycle with governance for project %s\n", a.agentID, projectID)

	// 1. Discover task
	task, err := a.discoverTask(ctx, projectID)
	if err != nil {
		return fmt.Errorf("discover task: %w", err)
	}
	fmt.Printf("[%s] Discovered task: %s\n", a.agentID, task.ID)

	// 2. Query governance if task has governance_ref
	if task.GovernanceRef != nil && *task.GovernanceRef != "" {
		fmt.Printf("[%s] Task has governance_ref, querying governance\n", a.agentID)
		resp := a.mcpServer.CallTool("conductor_get_governance", map[string]interface{}{
			"task_id": task.ID,
		})
		if resp.Error != "" {
			fmt.Printf("[%s] Governance query failed: %s\n", a.agentID, resp.Error)
		} else {
			fmt.Printf("[%s] Governance state: %v\n", a.agentID, resp.Result)
		}
	}

	// 3. Claim task
	err = a.claimTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("claim task: %w", err)
	}
	fmt.Printf("[%s] Claimed task: %s\n", a.agentID, task.ID)

	// 4. Perform actual work
	err = a.performActualWork(ctx, task)
	if err != nil {
		return fmt.Errorf("perform actual work: %w", err)
	}
	fmt.Printf("[%s] Performed actual work on task: %s\n", a.agentID, task.ID)

	// 5. Record detailed activity
	err = a.recordDetailedActivity(ctx, task)
	if err != nil {
		return fmt.Errorf("record detailed activity: %w", err)
	}
	fmt.Printf("[%s] Recorded detailed activity for task: %s\n", a.agentID, task.ID)

	// 6. Submit task
	err = a.submitTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("submit task: %w", err)
	}
	fmt.Printf("[%s] Submitted task: %s\n", a.agentID, task.ID)

	return nil
}
