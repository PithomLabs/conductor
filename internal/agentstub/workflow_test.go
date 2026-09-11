package agentstub

import (
	"context"
	"testing"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

func setupTestWorkflow(t *testing.T) (*store.DB, *governance.NullReader) {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &governance.NullReader{}
	return db, gov
}

func TestEndToEndWorkflow(t *testing.T) {
	db, gov := setupTestWorkflow(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)
	depRepo := store.NewDependencyRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-e2e",
		Name: "End-to-End Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Decompose work - create tasks
	err = taskRepo.Create(ctx, &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-e2e",
		Title:     "First Task",
		Status:    domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task-1 error: %v", err)
	}

	err = taskRepo.Create(ctx, &domain.Task{
		ID:        "task-2",
		ProjectID: "proj-e2e",
		Title:     "Second Task",
		Status:    domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task-2 error: %v", err)
	}

	// 3. Create dependency: task-1 blocks task-2
	err = depRepo.Create(ctx, &domain.Dependency{
		ID:          "dep-1",
		TaskID:      "task-2",
		BlockedByID: "task-1",
	})
	if err != nil {
		t.Fatalf("create dependency error: %v", err)
	}

	// 4. Verify dependencies exist
	deps, err := depRepo.ListByTask(ctx, "task-2")
	if err != nil {
		t.Fatalf("list dependencies error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	// 5. Agent claims and completes task-1
	stub := NewStub(db, gov, "agent-e2e")
	err = stub.Run(ctx, "proj-e2e")
	if err != nil {
		t.Fatalf("stub run error: %v", err)
	}

	// 6. Verify task-1 is in review
	task1, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task-1 error: %v", err)
	}
	if task1.Status != domain.TaskStatusReview {
		t.Errorf("expected task-1 status review, got %s", task1.Status)
	}

	// 7. Reviewer accepts task-1
	reviewer := NewReviewer(db, gov, "reviewer-e2e")
	err = reviewer.ReviewTask(ctx, "task-1", true)
	if err != nil {
		t.Fatalf("reviewer accept error: %v", err)
	}

	// 8. Verify task-1 is accepted
	task1, err = taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task-1 error: %v", err)
	}
	if task1.Status != domain.TaskStatusAccepted {
		t.Errorf("expected task-1 status accepted, got %s", task1.Status)
	}

	// 9. Verify task-2 is still proposed
	task2, err := taskRepo.GetByID(ctx, "task-2")
	if err != nil {
		t.Fatalf("get task-2 error: %v", err)
	}
	if task2.Status != domain.TaskStatusProposed {
		t.Errorf("expected task-2 status proposed, got %s", task2.Status)
	}
}

func TestMultiTaskWorkflow(t *testing.T) {
	db, gov := setupTestWorkflow(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-multi",
		Name: "Multi-Task Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create multiple tasks
	for i := 1; i <= 3; i++ {
		err = taskRepo.Create(ctx, &domain.Task{
			ID:        "task-" + string(rune('0'+i)),
			ProjectID: "proj-multi",
			Title:     "Task " + string(rune('0'+i)),
			Status:    domain.TaskStatusProposed,
		})
		if err != nil {
			t.Fatalf("create task-%d error: %v", i, err)
		}
	}

	// 3. Agent completes first task
	stub := NewStub(db, gov, "agent-multi")
	err = stub.Run(ctx, "proj-multi")
	if err != nil {
		t.Fatalf("stub run error: %v", err)
	}

	// 4. Verify first task is in review
	task1, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task-1 error: %v", err)
	}
	if task1.Status != domain.TaskStatusReview {
		t.Errorf("expected task-1 status review, got %s", task1.Status)
	}

	// 5. Reviewer rejects task-1
	reviewer := NewReviewer(db, gov, "reviewer-multi")
	err = reviewer.ReviewTask(ctx, "task-1", false)
	if err != nil {
		t.Fatalf("reviewer reject error: %v", err)
	}

	// 6. Verify task-1 is back to active
	task1, err = taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task-1 error: %v", err)
	}
	if task1.Status != domain.TaskStatusActive {
		t.Errorf("expected task-1 status active after reject, got %s", task1.Status)
	}
}

func TestGovernanceIntersection(t *testing.T) {
	db, gov := setupTestWorkflow(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-gov",
		Name: "Governance Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create task with governance_ref
	govRef := `{"provider":"solvent","reference_id":"scenario-gov"}`
	err = taskRepo.Create(ctx, &domain.Task{
		ID:            "task-gov",
		ProjectID:     "proj-gov",
		Title:         "Governed Task",
		Status:        domain.TaskStatusProposed,
		GovernanceRef: &govRef,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Agent completes task with governance
	stub := NewStub(db, gov, "agent-gov")
	err = stub.RunWithGovernance(ctx, "proj-gov")
	if err != nil {
		t.Fatalf("stub run error: %v", err)
	}

	// 4. Verify task is in review
	task, err := taskRepo.GetByID(ctx, "task-gov")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}

	// 5. Verify governance_ref is preserved
	if task.GovernanceRef == nil || *task.GovernanceRef != govRef {
		t.Errorf("expected governance_ref to be preserved")
	}
}

func TestCompleteWorkflowAllPhases(t *testing.T) {
	db, gov := setupTestWorkflow(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)
	activityRepo := store.NewActivityRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-all",
		Name: "Complete Workflow Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create task
	err = taskRepo.Create(ctx, &domain.Task{
		ID:        "task-all",
		ProjectID: "proj-all",
		Title:     "Complete Workflow Task",
		Status:    domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Agent completes full lifecycle
	stub := NewStub(db, gov, "agent-all")
	err = stub.Run(ctx, "proj-all")
	if err != nil {
		t.Fatalf("stub run error: %v", err)
	}

	// 4. Verify task is in review
	task, err := taskRepo.GetByID(ctx, "task-all")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}

	// 5. Verify agent is set
	if task.CurrentAgent == nil || *task.CurrentAgent != "agent-all" {
		t.Errorf("expected agent agent-all, got %v", task.CurrentAgent)
	}

	// 6. Reviewer accepts
	reviewer := NewReviewer(db, gov, "reviewer-all")
	err = reviewer.ReviewTask(ctx, "task-all", true)
	if err != nil {
		t.Fatalf("reviewer accept error: %v", err)
	}

	// 7. Verify task is accepted
	task, err = taskRepo.GetByID(ctx, "task-all")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusAccepted {
		t.Errorf("expected status accepted, got %s", task.Status)
	}

	// 8. Verify activity was recorded
	activities, err := activityRepo.ListByTask(ctx, "task-all")
	if err != nil {
		t.Fatalf("list activities error: %v", err)
	}
	if len(activities) == 0 {
		t.Errorf("expected activities to be recorded")
	}

	// 9. Verify no execution through Conductor
	// Task should be accepted through review, not through execution
	if task.Status != domain.TaskStatusAccepted {
		t.Errorf("task should be accepted through review")
	}
}
