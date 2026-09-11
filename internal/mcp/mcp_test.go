package mcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

func setupTestMCP(t *testing.T) *Server {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &governance.NullReader{}
	return NewServer(db, gov, "test-agent")
}

func TestMCPListTools(t *testing.T) {
	server := setupTestMCP(t)

	resp := server.handleListTools()
	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type: %T", resp.Result)
	}

	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatalf("unexpected tools type: %T", result["tools"])
	}

		expectedTools := []string{
		"conductor_release_task",
		"conductor_get_project",
		"conductor_list_tasks",
		"conductor_get_task",
		"conductor_create_task",
		"conductor_update_task",
		"conductor_claim_task",
		"conductor_submit_task",
		"conductor_accept_task",
		"conductor_reject_task",
		"conductor_report_blocker",
		"conductor_resolve_blocker",
		"conductor_post_activity",
		"conductor_get_governance",
		"conductor_next_task",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool["name"] == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tool %s not found", expected)
		}
	}
}

func TestMCPClaimTask(t *testing.T) {
	server := setupTestMCP(t)

	// Create project first (required for foreign key)
	err := server.projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// Create a task
	resp := server.handleTool("conductor_create_task", map[string]interface{}{
		"project_id": "proj-1",
		"title":      "Test Task",
		"task_id":    "task-1",
	})
	if resp.Error != "" {
		t.Fatalf("create task error: %s", resp.Error)
	}

	// Claim task
	resp = server.handleTool("conductor_claim_task", map[string]interface{}{
		"task_id": "task-1",
	})
	if resp.Error != "" {
		t.Fatalf("claim task error: %s", resp.Error)
	}
}

func TestMCPInvalidTool(t *testing.T) {
	server := setupTestMCP(t)

	resp := server.handleTool("invalid_tool", map[string]interface{}{})
	if resp.Error == "" {
		t.Error("expected error for invalid tool")
	}
}

type errorGovernanceReader struct{}

func (r *errorGovernanceReader) GetState(ctx context.Context, ref governance.GovernanceReference) (*governance.GovernanceState, error) {
	return nil, fmt.Errorf("connection refused")
}

func (r *errorGovernanceReader) CheckAuthorization(ctx context.Context, ref governance.GovernanceReference, action string) (*governance.AuthorizationResult, error) {
	return nil, fmt.Errorf("connection refused")
}

func TestMCPGetGovernanceProviderFailure(t *testing.T) {
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &errorGovernanceReader{}
	server := NewServer(db, gov, "test-agent")

	// Create project and task with governance_ref
	err = server.projectRepo.Create(ctx, &domain.Project{ID: "proj-1", Name: "Test"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	govRef := `{"provider":"solvent","reference_id":"scenario-1"}`
	err = server.taskRepo.Create(ctx, &domain.Task{
		ID:            "task-1",
		ProjectID:     "proj-1",
		Title:         "Test",
		GovernanceRef: &govRef,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	resp := server.handleTool("conductor_get_governance", map[string]interface{}{
		"task_id": "task-1",
	})
	if resp.Error != "" {
		t.Fatalf("expected no error, got: %s", resp.Error)
	}

	state, ok := resp.Result.(*governance.GovernanceState)
	if !ok {
		t.Fatalf("expected *GovernanceState, got %T", resp.Result)
	}
	if state.Status != governance.GovernanceStatusUnknown {
		t.Errorf("expected status unknown, got %s", state.Status)
	}
	if len(state.Blockers) == 0 || state.Blockers[0] != "provider unavailable" {
		t.Errorf("expected 'provider unavailable' blocker, got %v", state.Blockers)
	}
}
