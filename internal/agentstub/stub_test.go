package agentstub

import (
	"context"
	"testing"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

func setupTestStub(t *testing.T) (*store.DB, *governance.NullReader) {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &governance.NullReader{}
	return db, gov
}

func TestStubCompletesLifecycle(t *testing.T) {
	db, gov := setupTestStub(t)
	ctx := context.Background()

	err := store.NewProjectRepository(db).Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	err = store.NewTaskRepository(db).Create(ctx, &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	stub := NewStub(db, gov, "test-agent")
	err = stub.Run(ctx, "proj-1")
	if err != nil {
		t.Fatalf("stub run error: %v", err)
	}

	task, err := store.NewTaskRepository(db).GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}

	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}

	if task.CurrentAgent == nil || *task.CurrentAgent != "test-agent" {
		t.Errorf("expected agent test-agent, got %v", task.CurrentAgent)
	}
}

func TestStubWithGovernance(t *testing.T) {
	db, gov := setupTestStub(t)
	ctx := context.Background()

	err := store.NewProjectRepository(db).Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	govRef := `{"provider":"solvent","reference_id":"scenario-1"}`
	err = store.NewTaskRepository(db).Create(ctx, &domain.Task{
		ID:            "task-1",
		ProjectID:     "proj-1",
		Title:         "Test Task",
		GovernanceRef: &govRef,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	stub := NewStub(db, gov, "test-agent")
	err = stub.RunWithGovernance(ctx, "proj-1")
	if err != nil {
		t.Fatalf("stub run error: %v", err)
	}

	task, err := store.NewTaskRepository(db).GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}

	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}
}

func TestReviewerAcceptsTask(t *testing.T) {
	db, gov := setupTestStub(t)
	ctx := context.Background()

	err := store.NewProjectRepository(db).Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	err = store.NewTaskRepository(db).Create(ctx, &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	err = store.NewTaskRepository(db).Claim(ctx, "task-1", "agent-1")
	if err != nil {
		t.Fatalf("claim error: %v", err)
	}
	err = store.NewTaskRepository(db).Transition(ctx, "task-1", domain.TaskStatusActive, domain.TaskStatusReview, "agent", "agent-1", "task.submitted", "{}")
	if err != nil {
		t.Fatalf("submit error: %v", err)
	}

	reviewer := NewReviewer(db, gov, "reviewer-1")
	err = reviewer.ReviewTask(ctx, "task-1", true)
	if err != nil {
		t.Fatalf("reviewer accept error: %v", err)
	}

	task, err := store.NewTaskRepository(db).GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}

	if task.Status != domain.TaskStatusAccepted {
		t.Errorf("expected status accepted, got %s", task.Status)
	}
}

func TestReviewerRejectsTask(t *testing.T) {
	db, gov := setupTestStub(t)
	ctx := context.Background()

	err := store.NewProjectRepository(db).Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	err = store.NewTaskRepository(db).Create(ctx, &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	err = store.NewTaskRepository(db).Claim(ctx, "task-1", "agent-1")
	if err != nil {
		t.Fatalf("claim error: %v", err)
	}
	err = store.NewTaskRepository(db).Transition(ctx, "task-1", domain.TaskStatusActive, domain.TaskStatusReview, "agent", "agent-1", "task.submitted", "{}")
	if err != nil {
		t.Fatalf("submit error: %v", err)
	}

	reviewer := NewReviewer(db, gov, "reviewer-1")
	err = reviewer.ReviewTask(ctx, "task-1", false)
	if err != nil {
		t.Fatalf("reviewer reject error: %v", err)
	}

	task, err := store.NewTaskRepository(db).GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}

	if task.Status != domain.TaskStatusActive {
		t.Errorf("expected status active, got %s", task.Status)
	}
}

func TestStubDoesNotExecute(t *testing.T) {
	db, gov := setupTestStub(t)
	ctx := context.Background()

	err := store.NewProjectRepository(db).Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	err = store.NewTaskRepository(db).Create(ctx, &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	stub := NewStub(db, gov, "test-agent")
	err = stub.Run(ctx, "proj-1")
	if err != nil {
		t.Fatalf("stub run error: %v", err)
	}

	task, err := store.NewTaskRepository(db).GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}

	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review (no execution through Conductor), got %s", task.Status)
	}

	if task.Status == domain.TaskStatusAccepted {
		t.Error("task should not be accepted through Conductor (execution should happen externally)")
	}
}
