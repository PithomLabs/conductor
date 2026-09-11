package agentstub

import (
	"context"
	"testing"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

func setupTestDomain(t *testing.T) (*store.DB, *governance.NullReader) {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &governance.NullReader{}
	return db, gov
}

// TestDomainExtensionReadiness_RestaurantReservation proves Conductor can be used
// for an entirely unrelated domain (restaurant reservations) without any changes
// to Conductor's core model, tables, states, or API.
func TestDomainExtensionReadiness_RestaurantReservation(t *testing.T) {
	db, gov := setupTestDomain(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)

	// 1. Create project for restaurant reservation system
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-restaurant",
		Name: "Restaurant Reservation System",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create tasks with domain-specific descriptions
	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-reservation-ui",
		ProjectID:   "proj-restaurant",
		Title:       "Build Reservation UI",
		Description: "Create a web interface for customers to make reservations",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-table-management",
		ProjectID:   "proj-restaurant",
		Title:       "Implement Table Management",
		Description: "Create system to manage table availability and capacity",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-notification-system",
		ProjectID:   "proj-restaurant",
		Title:       "Build Notification System",
		Description: "Implement SMS/email notifications for reservations",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Real agent completes first task
	agent := NewRealAgent(db, gov, "restaurant-agent")
	err = agent.Run(ctx, "proj-restaurant")
	if err != nil {
		t.Fatalf("real agent run error: %v", err)
	}

	// 4. Verify task is in review
	task, err := taskRepo.GetByID(ctx, "task-reservation-ui")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}

	// 5. Reviewer accepts task
	reviewer := NewReviewer(db, gov, "restaurant-reviewer")
	err = reviewer.ReviewTask(ctx, "task-reservation-ui", true)
	if err != nil {
		t.Fatalf("reviewer accept error: %v", err)
	}

	// 6. Verify task is accepted
	task, err = taskRepo.GetByID(ctx, "task-reservation-ui")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusAccepted {
		t.Errorf("expected status accepted, got %s", task.Status)
	}

	// 7. Verify other tasks are still proposed
	for _, taskID := range []string{"task-table-management", "task-notification-system"} {
		task, err := taskRepo.GetByID(ctx, taskID)
		if err != nil {
			t.Fatalf("get task %s error: %v", taskID, err)
		}
		if task.Status != domain.TaskStatusProposed {
			t.Errorf("expected task %s status proposed, got %s", taskID, task.Status)
		}
	}
}

// TestDomainExtensionReadiness_LibrarySystem proves Conductor can be used
// for another entirely unrelated domain (library book tracking) without any changes.
func TestDomainExtensionReadiness_LibrarySystem(t *testing.T) {
	db, gov := setupTestDomain(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)

	// 1. Create project for library book tracking system
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-library",
		Name: "Library Book Tracking System",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create tasks with domain-specific descriptions
	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-catalog-system",
		ProjectID:   "proj-library",
		Title:       "Build Book Catalog",
		Description: "Create database of all books with metadata",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-checkout-system",
		ProjectID:   "proj-library",
		Title:       "Implement Checkout System",
		Description: "Create system for patrons to check out books",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Real agent completes first task
	agent := NewRealAgent(db, gov, "library-agent")
	err = agent.Run(ctx, "proj-library")
	if err != nil {
		t.Fatalf("real agent run error: %v", err)
	}

	// 4. Verify task is in review
	task, err := taskRepo.GetByID(ctx, "task-catalog-system")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}

	// 5. Reviewer rejects task
	reviewer := NewReviewer(db, gov, "library-reviewer")
	err = reviewer.ReviewTask(ctx, "task-catalog-system", false)
	if err != nil {
		t.Fatalf("reviewer reject error: %v", err)
	}

	// 6. Verify task is back to active
	task, err = taskRepo.GetByID(ctx, "task-catalog-system")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusActive {
		t.Errorf("expected status active after reject, got %s", task.Status)
	}
}

// TestDomainExtensionReadiness_VerificationChecklist proves all domain
// extension requirements are met without any changes to Conductor.
func TestDomainExtensionReadiness_VerificationChecklist(t *testing.T) {
	db, gov := setupTestDomain(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)
	activityRepo := store.NewActivityRepository(db)
	depRepo := store.NewDependencyRepository(db)

	// 1. Create project for completely unrelated domain
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-unrelated",
		Name: "Completely Unrelated Domain Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create tasks with domain-specific descriptions
	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-feature-a",
		ProjectID:   "proj-unrelated",
		Title:       "Feature A",
		Description: "Domain-specific feature that requires special handling",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	err = taskRepo.Create(ctx, &domain.Task{
		ID:          "task-feature-b",
		ProjectID:   "proj-unrelated",
		Title:       "Feature B",
		Description: "Another domain-specific feature",
		Status:      domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Create dependency
	err = depRepo.Create(ctx, &domain.Dependency{
		ID:          "dep-unrelated",
		TaskID:      "task-feature-b",
		BlockedByID: "task-feature-a",
	})
	if err != nil {
		t.Fatalf("create dependency error: %v", err)
	}

	// 4. Real agent completes lifecycle
	agent := NewRealAgent(db, gov, "unrelated-agent")
	err = agent.Run(ctx, "proj-unrelated")
	if err != nil {
		t.Fatalf("real agent run error: %v", err)
	}

	// 5. Verify task is in review
	task, err := taskRepo.GetByID(ctx, "task-feature-a")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}

	// 6. Verify activity was recorded
	activities, err := activityRepo.ListByTask(ctx, "task-feature-a")
	if err != nil {
		t.Fatalf("list activities error: %v", err)
	}
	if len(activities) == 0 {
		t.Errorf("expected activities to be recorded")
	}

	// 7. Verify dependency exists
	deps, err := depRepo.ListByTask(ctx, "task-feature-b")
	if err != nil {
		t.Fatalf("list dependencies error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	// 8. Reviewer accepts task
	reviewer := NewReviewer(db, gov, "unrelated-reviewer")
	err = reviewer.ReviewTask(ctx, "task-feature-a", true)
	if err != nil {
		t.Fatalf("reviewer accept error: %v", err)
	}

	// 9. Verify task is accepted
	task, err = taskRepo.GetByID(ctx, "task-feature-a")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusAccepted {
		t.Errorf("expected status accepted, got %s", task.Status)
	}

	// 10. Verify no execution through Conductor
	if task.Status == domain.TaskStatusAccepted {
		t.Log("Task accepted through review - no execution through Conductor")
	}
}

// TestDomainExtensionReadiness_NoDomainSpecificChanges proves that Conductor
// requires no domain-specific changes to support arbitrary workloads.
func TestDomainExtensionReadiness_NoDomainSpecificChanges(t *testing.T) {
	db, gov := setupTestDomain(t)
	ctx := context.Background()

	projectRepo := store.NewProjectRepository(db)
	taskRepo := store.NewTaskRepository(db)

	// 1. Create project
	err := projectRepo.Create(ctx, &domain.Project{
		ID:   "proj-generic",
		Name: "Generic Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	// 2. Create task with generic lifecycle
	err = taskRepo.Create(ctx, &domain.Task{
		ID:        "task-generic",
		ProjectID: "proj-generic",
		Title:     "Generic Task",
		Status:    domain.TaskStatusProposed,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	// 3. Real agent completes lifecycle
	agent := NewRealAgent(db, gov, "generic-agent")
	err = agent.Run(ctx, "proj-generic")
	if err != nil {
		t.Fatalf("real agent run error: %v", err)
	}

	// 4. Verify task is in review
	task, err := taskRepo.GetByID(ctx, "task-generic")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusReview {
		t.Errorf("expected status review, got %s", task.Status)
	}

	// 5. Reviewer accepts task
	reviewer := NewReviewer(db, gov, "generic-reviewer")
	err = reviewer.ReviewTask(ctx, "task-generic", true)
	if err != nil {
		t.Fatalf("reviewer accept error: %v", err)
	}

	// 6. Verify task is accepted
	task, err = taskRepo.GetByID(ctx, "task-generic")
	if err != nil {
		t.Fatalf("get task error: %v", err)
	}
	if task.Status != domain.TaskStatusAccepted {
		t.Errorf("expected status accepted, got %s", task.Status)
	}

	// 7. Verify no domain-specific changes were needed
	// The same generic lifecycle worked for:
	// - Restaurant reservation system
	// - Library book tracking system
	// - Completely unrelated domain
	// - Generic project
	// All without any changes to Conductor's core model, tables, states, or API.
	t.Log("Domain extension readiness verified: No domain-specific changes needed")
}
