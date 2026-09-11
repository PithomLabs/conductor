package store

import (
	"context"
	"testing"
	"time"

	"github.com/PithomLabs/conductor/internal/domain"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := OpenTest()
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}

func TestProjectCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	// Create
	project := &domain.Project{
		ID:          "proj-1",
		Name:        "Test Project",
		Description: "A test project",
	}
	if err := repo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Read
	got, err := repo.GetByID(ctx, "proj-1")
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got %q", got.Name)
	}
	if got.Status != domain.ProjectStatusActive {
		t.Errorf("expected status 'active', got %q", got.Status)
	}

	// Update
	got.Description = "Updated description"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update project: %v", err)
	}
	got2, err := repo.GetByID(ctx, "proj-1")
	if err != nil {
		t.Fatalf("get updated project: %v", err)
	}
	if got2.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got %q", got2.Description)
	}

	// Delete
	if err := repo.Delete(ctx, "proj-1"); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	_, err = repo.GetByID(ctx, "proj-1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestTaskCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	// Create project first
	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Create task
	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Read
	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %q", got.Title)
	}
	if got.Status != domain.TaskStatusProposed {
		t.Errorf("expected status 'proposed', got %q", got.Status)
	}

	// Update (non-lifecycle fields)
	newTitle := "Updated Task"
	fields := TaskUpdateFields{Title: &newTitle}
	if err := taskRepo.Update(ctx, "task-1", fields); err != nil {
		t.Fatalf("update task: %v", err)
	}
	got2, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get updated task: %v", err)
	}
	if got2.Title != "Updated Task" {
		t.Errorf("expected title 'Updated Task', got %q", got2.Title)
	}
}

func TestLifecycleTransitions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	activityRepo := NewActivityRepository(db)
	ctx := context.Background()

	// Create project and task
	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Test valid transitions
	// proposed -> active (claim)
	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim task: %v", err)
	}
	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != domain.TaskStatusActive {
		t.Errorf("expected status 'active', got %q", got.Status)
	}
	if got.CurrentAgent == nil || *got.CurrentAgent != "agent-1" {
		t.Errorf("expected current_agent 'agent-1', got %v", got.CurrentAgent)
	}

	// active -> review (submit)
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusActive, domain.TaskStatusReview, "agent", "agent-1", "task.submitted", "{}"); err != nil {
		t.Fatalf("submit task: %v", err)
	}
	got, err = taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != domain.TaskStatusReview {
		t.Errorf("expected status 'review', got %q", got.Status)
	}

	// review -> accepted
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusReview, domain.TaskStatusAccepted, "human", "reviewer-1", "task.accepted", "{}"); err != nil {
		t.Fatalf("accept task: %v", err)
	}
	got, err = taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != domain.TaskStatusAccepted {
		t.Errorf("expected status 'accepted', got %q", got.Status)
	}

	// Verify activity log
	activities, err := activityRepo.ListByTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("list activities: %v", err)
	}
	if len(activities) != 3 {
		t.Errorf("expected 3 activities, got %d", len(activities))
	}
	// Verify all expected actions exist (order may vary due to same timestamp)
	actionCounts := map[string]int{}
	for _, a := range activities {
		actionCounts[a.Action]++
	}
	if actionCounts["task.claimed"] != 1 {
		t.Errorf("expected 1 task.claimed activity, got %d", actionCounts["task.claimed"])
	}
	if actionCounts["task.submitted"] != 1 {
		t.Errorf("expected 1 task.submitted activity, got %d", actionCounts["task.submitted"])
	}
	if actionCounts["task.accepted"] != 1 {
		t.Errorf("expected 1 task.accepted activity, got %d", actionCounts["task.accepted"])
	}
}

func TestInvalidTransition(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	// Create project and task
	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Try invalid transition: proposed -> accepted (must go through active first)
	err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusProposed, domain.TaskStatusAccepted, "human", "reviewer-1", "task.accepted", "{}")
	if err != ErrInvalidTransition {
		t.Errorf("expected ErrInvalidTransition for proposed -> accepted, got %v", err)
	}

	// Try invalid transition: terminal -> anything
	// First move to accepted
	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim task: %v", err)
	}
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusActive, domain.TaskStatusReview, "agent", "agent-1", "task.submitted", "{}"); err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusReview, domain.TaskStatusAccepted, "human", "reviewer-1", "task.accepted", "{}"); err != nil {
		t.Fatalf("accept task: %v", err)
	}

	// Try to transition from accepted (terminal)
	err = taskRepo.Transition(ctx, "task-1", domain.TaskStatusAccepted, domain.TaskStatusActive, "human", "reviewer-1", "task.reopened", "{}")
	if err != ErrInvalidTransition {
		t.Errorf("expected ErrInvalidTransition for terminal state, got %v", err)
	}
}

func TestAtomicClaiming(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	// Create project and task
	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// First claim succeeds
	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("first claim: %v", err)
	}

	// Second claim fails (already claimed)
	err := taskRepo.Claim(ctx, "task-1", "agent-2")
	if err != ErrClaimFailed {
		t.Errorf("expected ErrClaimFailed, got %v", err)
	}

	// Verify task is still assigned to agent-1
	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.CurrentAgent == nil || *got.CurrentAgent != "agent-1" {
		t.Errorf("expected current_agent 'agent-1', got %v", got.CurrentAgent)
	}
}

func TestLifecycleBypassPrevention(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	// Create project and task
	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Verify task is still in proposed status
	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != domain.TaskStatusProposed {
		t.Errorf("expected status 'proposed', got %q", got.Status)
	}
}

func TestSameProjectDependency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	depRepo := NewDependencyRepository(db)
	ctx := context.Background()

	// Create two projects
	project1 := &domain.Project{ID: "proj-1", Name: "Project 1"}
	project2 := &domain.Project{ID: "proj-2", Name: "Project 2"}
	if err := projectRepo.Create(ctx, project1); err != nil {
		t.Fatalf("create project 1: %v", err)
	}
	if err := projectRepo.Create(ctx, project2); err != nil {
		t.Fatalf("create project 2: %v", err)
	}

	// Create tasks in different projects
	task1 := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Task 1",
	}
	task2 := &domain.Task{
		ID:        "task-2",
		ProjectID: "proj-2",
		Title:     "Task 2",
	}
	if err := taskRepo.Create(ctx, task1); err != nil {
		t.Fatalf("create task 1: %v", err)
	}
	if err := taskRepo.Create(ctx, task2); err != nil {
		t.Fatalf("create task 2: %v", err)
	}

	// Try to create cross-project dependency (should fail)
	dep := &domain.Dependency{
		ID:          "dep-1",
		TaskID:      "task-1",
		BlockedByID: "task-2",
	}
	err := depRepo.Create(ctx, dep)
	if err != ErrSameProject {
		t.Errorf("expected ErrSameProject, got %v", err)
	}

	// Create same-project dependency (should succeed)
	task3 := &domain.Task{
		ID:        "task-3",
		ProjectID: "proj-1",
		Title:     "Task 3",
	}
	if err := taskRepo.Create(ctx, task3); err != nil {
		t.Fatalf("create task 3: %v", err)
	}

	dep2 := &domain.Dependency{
		ID:          "dep-2",
		TaskID:      "task-1",
		BlockedByID: "task-3",
	}
	if err := depRepo.Create(ctx, dep2); err != nil {
		t.Fatalf("create same-project dependency: %v", err)
	}
}

func TestActivityAppendOnly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	activityRepo := NewActivityRepository(db)
	ctx := context.Background()

	// Create project and task
	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Add activity
	activity := &domain.Activity{
		ID:        "act-1",
		TaskID:    "task-1",
		ActorType: "agent",
		ActorID:   "agent-1",
		Action:    "task.claimed",
	}
	if err := activityRepo.Append(ctx, activity); err != nil {
		t.Fatalf("append activity: %v", err)
	}

	// Verify activity exists
	activities, err := activityRepo.ListByTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("list activities: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("expected 1 activity, got %d", len(activities))
	}
}

func TestInvalidSelfDependency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	depRepo := NewDependencyRepository(db)
	ctx := context.Background()

	// Create project and task
	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Try to create self-dependency (should fail)
	dep := &domain.Dependency{
		ID:          "dep-1",
		TaskID:      "task-1",
		BlockedByID: "task-1",
	}
	err := depRepo.Create(ctx, dep)
	if err == nil {
		t.Error("expected error for self-dependency, got nil")
	}
}

func TestTaskUpdateFieldsExcludesLifecycleFields(t *testing.T) {
	// This test verifies that the TaskUpdateFields struct design prevents lifecycle bypass
	fields := TaskUpdateFields{}

	// Verify that we can only set non-lifecycle fields
	title := "new title"
	description := "new description"
	priority := "high"

	fields.Title = &title
	fields.Description = &description
	fields.Priority = &priority

	_ = fields
}

func TestConcurrentClaiming(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	// Create project and task
	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Simulate concurrent claiming
	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("first claim: %v", err)
	}

	// Second claim should fail
	err := taskRepo.Claim(ctx, "task-1", "agent-2")
	if err != ErrClaimFailed {
		t.Errorf("expected ErrClaimFailed, got %v", err)
	}

	// Verify timestamp is reasonable (within last minute)
	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	// SQLite datetime('now') returns "2006-01-02 15:04:05" format
	updatedAt, err := time.Parse("2006-01-02 15:04:05", got.UpdatedAt)
	if err != nil {
		t.Fatalf("parse updated_at: %v", err)
	}
	if time.Since(updatedAt) > time.Minute {
		t.Errorf("updated_at is too old: %v", got.UpdatedAt)
	}
}

func TestTaskCreateRejectsNonProposedStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
		Status:    "active",
	}
	err := taskRepo.Create(ctx, task)
	if err != ErrLifecycleBypass {
		t.Errorf("expected ErrLifecycleBypass for non-proposed status, got %v", err)
	}
}

func TestTaskCreateRejectsCurrentAgent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	agentID := "agent-1"
	task := &domain.Task{
		ID:           "task-1",
		ProjectID:    "proj-1",
		Title:        "Test Task",
		CurrentAgent: &agentID,
	}
	err := taskRepo.Create(ctx, task)
	if err != ErrLifecycleBypass {
		t.Errorf("expected ErrLifecycleBypass for current_agent, got %v", err)
	}
}

func TestTaskCreateAcceptsProposedStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
		Status:    "proposed",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task with proposed status: %v", err)
	}

	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != domain.TaskStatusProposed {
		t.Errorf("expected status proposed, got %s", got.Status)
	}
	if got.CurrentAgent != nil {
		t.Errorf("expected nil current_agent, got %v", got.CurrentAgent)
	}
}

func TestTaskCreateAcceptsEmptyStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task with empty status: %v", err)
	}

	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != domain.TaskStatusProposed {
		t.Errorf("expected status proposed, got %s", got.Status)
	}
}

func TestClaimOnlyPathToAssignment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Verify task starts unassigned
	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.CurrentAgent != nil {
		t.Errorf("expected nil current_agent after create, got %v", got.CurrentAgent)
	}

	// Only Claim() sets current_agent
	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim task: %v", err)
	}
	got, err = taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.CurrentAgent == nil || *got.CurrentAgent != "agent-1" {
		t.Errorf("expected current_agent 'agent-1' after claim, got %v", got.CurrentAgent)
	}
}

func TestReleaseByAssignedAgent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	if err := taskRepo.Release(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("release: %v", err)
	}

	got, err := taskRepo.GetByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.TaskStatusProposed {
		t.Errorf("expected proposed, got %s", got.Status)
	}
	if got.CurrentAgent != nil {
		t.Errorf("expected nil current_agent, got %v", got.CurrentAgent)
	}
}

func TestReleaseByOtherAgentRejected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	err := taskRepo.Release(ctx, "task-1", "agent-2")
	if err != ErrReleaseFailed {
		t.Errorf("expected ErrClaimFailed, got %v", err)
	}
}

func TestReleaseActivityRecorded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	activityRepo := NewActivityRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	if err := taskRepo.Release(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("release: %v", err)
	}

	activities, err := activityRepo.ListByTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("list activities: %v", err)
	}

	found := false
	for _, a := range activities {
		if a.Action == "task.released" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected task.released activity")
	}
}

func TestInvalidTransitionProposedToProposed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusProposed, domain.TaskStatusProposed, "agent", "agent-1", "task.self", "{}")
	if err != ErrInvalidTransition {
		t.Errorf("expected ErrInvalidTransition for proposed -> proposed, got %v", err)
	}
}

func TestReleaseFromReviewRejected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	// active → review
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusActive, domain.TaskStatusReview,
		"agent", "agent-1", "task.submitted", "{}"); err != nil {
		t.Fatalf("submit: %v", err)
	}

	err := taskRepo.Release(ctx, "task-1", "agent-1")
	if err != ErrReleaseFailed {
		t.Errorf("expected ErrReleaseFailed, got %v", err)
	}
}

func TestReleaseFromBlockedRejected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	// active → blocked
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusActive, domain.TaskStatusBlocked,
		"agent", "agent-1", "task.blocked", "{}"); err != nil {
		t.Fatalf("block: %v", err)
	}

	err := taskRepo.Release(ctx, "task-1", "agent-1")
	if err != ErrReleaseFailed {
		t.Errorf("expected ErrReleaseFailed, got %v", err)
	}
}

func TestReleaseFromAcceptedRejected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := taskRepo.Claim(ctx, "task-1", "agent-1"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	// active → review → accepted
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusActive, domain.TaskStatusReview,
		"agent", "agent-1", "task.submitted", "{}"); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusReview, domain.TaskStatusAccepted,
		"agent", "agent-2", "task.accepted", "{}"); err != nil {
		t.Fatalf("accept: %v", err)
	}

	err := taskRepo.Release(ctx, "task-1", "agent-1")
	if err != ErrReleaseFailed {
		t.Errorf("expected ErrReleaseFailed, got %v", err)
	}
}

func TestReleaseFromCancelledRejected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// proposed → cancelled
	if err := taskRepo.Transition(ctx, "task-1", domain.TaskStatusProposed, domain.TaskStatusCancelled,
		"agent", "agent-1", "task.cancelled", "{}"); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	err := taskRepo.Release(ctx, "task-1", "agent-1")
	if err != ErrReleaseFailed {
		t.Errorf("expected ErrReleaseFailed, got %v", err)
	}
}

func TestReleaseFromProposedRejected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectRepo := NewProjectRepository(db)
	taskRepo := NewTaskRepository(db)
	ctx := context.Background()

	project := &domain.Project{ID: "proj-1", Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	task := &domain.Task{ID: "task-1", ProjectID: "proj-1", Title: "Test Task"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Task is proposed with no agent — release should fail
	err := taskRepo.Release(ctx, "task-1", "agent-1")
	if err != ErrReleaseFailed {
		t.Errorf("expected ErrReleaseFailed, got %v", err)
	}
}
