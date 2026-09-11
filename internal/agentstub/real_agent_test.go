package agentstub

import (
	"context"
	"testing"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

func setupTestRealAgent(t *testing.T) (*store.DB, *governance.NullReader) {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &governance.NullReader{}
	return db, gov
}

func TestRealAgentCompletesLifecycle(t *testing.T) {
	db, gov := setupTestRealAgent(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)
	activityRepo := store.NewActivityRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-real",
		Name: "Real Agent Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create task with real description
	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-real",
		ProjectID:   "proj-real",
		Title:       "Implement Feature X",
		Description: "Write a function that calculates Fibonacci numbers efficiently",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Real agent completes lifecycle
	agent := NewRealAgent(db, gov, "real-agent-1")
	err = agent.Run(ctx, "proj-real")
	if err != nil {
		t.Fatalf("real agent run error: %v", err)
	}

	// 4. Verify task is in review
	task, err := taskRepo.GetByID(ctx, "task-real")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}

	// 5. Verify agent is set
	if task.CurrentAgent == nil || *task.CurrentAgent != "real-agent-1" {
		t.Errorf("expected agent real-agent-1, got %v", task.CurrentAgent)
	}

	// 6. Verify activity was recorded
	activities, err := activityRepo.ListByTask(ctx, "task-real")
	if err != nil {
		t.Fatalf("list activities error: %v", err)
	}
	if len(activities) == 0 {
		t.Errorf("expected activities to be recorded")
	}

	// 7. Verify activity has detailed information
	foundWorkCompleted := false
	for _, activity := range activities {
		if activity.ActorID != "real-agent-1" {
			t.Errorf("expected actor real-agent-1, got %s", activity.ActorID)
		}
		if activity.Action == "work.completed" {
			foundWorkCompleted = true
		}
	}
	if !foundWorkCompleted {
		t.Errorf("expected work.completed activity to be recorded")
	}
}

func TestRealAgentWithGovernance(t *testing.T) {
	db, gov := setupTestRealAgent(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-real-gov",
		Name: "Real Agent Governance Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create task with governance_ref
	govRef := `{"provider":"solvent","reference_id":"scenario-real-gov"}`
	err = taskRepo.Create(ctx, &domain.Task{
		ID:            "task-real-gov",
		ProjectID:     "proj-real-gov",
		Title:         "Governed Real Task",
		Description:   "Task that requires governance observation",
		Status:        domain.TaskStatusProposed,
		GovernanceRef: &govRef,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Real agent completes lifecycle with governance
	agent := NewRealAgent(db, gov, "real-agent-gov")
	err = agent.RunWithGovernance(ctx, "proj-real-gov")
	if err != nil {
		t.Fatalf("real agent run error: %v", err)
	}

	// 4. Verify task is in review
	task, err := taskRepo.GetByID(ctx, "task-real-gov")
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

func TestRealAgentDoesNotExecute(t *testing.T) {
	db, gov := setupTestRealAgent(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-real-noexec",
		Name: "Real Agent No Execution Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create task
	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-real-noexec",
		ProjectID:   "proj-real-noexec",
		Title:       "Task Without Execution",
		Description: "Task that should not be executed through Conductor",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Real agent completes lifecycle
	agent := NewRealAgent(db, gov, "real-agent-noexec")
	err = agent.Run(ctx, "proj-real-noexec")
	if err != nil {
		t.Fatalf("real agent run error: %v", err)
	}

	// 4. Verify task is in review (not accepted - no execution through Conductor)
	task, err := taskRepo.GetByID(ctx, "task-real-noexec")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review (no execution through Conductor), got %s", task.Status)
	}

	// 5. Verify no execution happened through Conductor
	if task.Status == domain.TaskStatusAccepted {
		t.Error("task should not be accepted through Conductor (execution should happen externally)")
	}
}

func TestRealAgentMultipleTasks(t *testing.T) {
	db, gov := setupTestRealAgent(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-real-multi",
		Name: "Real Agent Multiple Tasks Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create multiple tasks
	for i := 1; i <= 3; i++ {
		err = taskRepo.Create(ctx, &domain.Task{
			ID:          "task-real-" + string(rune('0'+i)),
			ProjectID:   "proj-real-multi",
			Title:       "Real Task " + string(rune('0'+i)),
			Description: "Description for real task " + string(rune('0'+i)),
			Status:      domain.TaskStatusProposed,
		})
		if err != nil {
			t.Fatalf("create task-%d error: %v", i, err)
		}
	}

	// 3. Real agent completes first task
	agent := NewRealAgent(db, gov, "real-agent-multi")
	err = agent.Run(ctx, "proj-real-multi")
	if err != nil {
		t.Fatalf("real agent run error: %v", err)
	}

	// 4. Verify first task is in review
	task1, err := taskRepo.GetByID(ctx, "task-real-1")
	if err != nil {
		t.Fatalf("get task-1 error: %v", err)
	}
	if task1.Status != domain.TaskStatusReview {
		t.Errorf("expected task-1 status review, got %s", task1.Status)
	}

	// 5. Verify other tasks are still proposed
	for i := 2; i <= 3; i++ {
		task, err := taskRepo.GetByID(ctx, "task-real-"+string(rune('0'+i)))
		if err != nil {
			t.Fatalf("get task-%d error: %v", i, err)
		}
		if task.Status != domain.TaskStatusProposed {
			t.Errorf("expected task-%d status proposed, got %s", i, task.Status)
		}
	}
}
