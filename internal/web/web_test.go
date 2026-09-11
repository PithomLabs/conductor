package web

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

var ctx = context.Background()

func setupTestWeb(t *testing.T) *Server {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &governance.NullReader{}
	server := NewServer(":0", db, gov)

	server.templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	return server
}

func TestDashboardEmpty(t *testing.T) {
	server := setupTestWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	server.handleDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "No projects found") {
		t.Error("expected 'No projects found' message")
	}
}

func TestDashboardWithProjects(t *testing.T) {
	server := setupTestWeb(t)

	err := server.ProjectRepo.Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	server.handleDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test Project") {
		t.Error("expected 'Test Project' in response")
	}
}

func TestProjectView(t *testing.T) {
	server := setupTestWeb(t)

	err := server.ProjectRepo.Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	err = server.TaskRepo.Create(ctx, &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/project/proj-1", nil)
	w := httptest.NewRecorder()
	server.handleProject(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test Project") {
		t.Error("expected 'Test Project' in response")
	}
	if !strings.Contains(body, "Test Task") {
		t.Error("expected 'Test Task' in response")
	}
}

func TestTaskView(t *testing.T) {
	server := setupTestWeb(t)

	err := server.ProjectRepo.Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	err = server.TaskRepo.Create(ctx, &domain.Task{
		ID:        "task-1",
		ProjectID: "proj-1",
		Title:     "Test Task",
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	err = server.TaskRepo.Claim(ctx, "task-1", "agent-1")
	if err != nil {
		t.Fatalf("claim task error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/task/task-1", nil)
	w := httptest.NewRecorder()
	server.handleTask(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test Task") {
		t.Error("expected 'Test Task' in response")
	}
}

func TestTaskViewWithGovernance(t *testing.T) {
	server := setupTestWeb(t)

	err := server.ProjectRepo.Create(ctx, &domain.Project{
		ID:   "proj-1",
		Name: "Test Project",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}

	govRef := `{"provider":"solvent","reference_id":"scenario-1"}`
	err = server.TaskRepo.Create(ctx, &domain.Task{
		ID:            "task-1",
		ProjectID:     "proj-1",
		Title:         "Test Task",
		GovernanceRef: &govRef,
	})
	if err != nil {
		t.Fatalf("create task error: %v", err)
	}

	err = server.TaskRepo.Claim(ctx, "task-1", "agent-1")
	if err != nil {
		t.Fatalf("claim task error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/task/task-1", nil)
	w := httptest.NewRecorder()
	server.handleTask(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Governance (read-only)") {
		t.Error("expected 'Governance (read-only)' in response")
	}
	if !strings.Contains(body, "This is what the external governance system currently reports") {
		t.Error("expected observational note in governance panel")
	}
	if !strings.Contains(body, "As of") {
		t.Error("expected 'As of' timestamp in governance panel")
	}
}

func TestWebDefaultsToLocalBinding(t *testing.T) {
	server := NewServer("127.0.0.1:8080", nil, nil)
	if server.Addr != "127.0.0.1:8080" {
		t.Errorf("expected 127.0.0.1:8080, got %s", server.Addr)
	}
}


// fixedTimeReader returns a governance state with a fixed RefreshedAt for deterministic assertions.
type fixedTimeReader struct {
	refreshedAt time.Time
}

func (r *fixedTimeReader) GetState(ctx context.Context, ref governance.GovernanceReference) (*governance.GovernanceState, error) {
	return &governance.GovernanceState{
		Reference:   ref,
		Status:      governance.GovernanceStatusReady,
		RefreshedAt: r.refreshedAt,
	}, nil
}

// zeroTimeReader returns a governance state with zero RefreshedAt.
type zeroTimeReader struct{}

func (r *zeroTimeReader) GetState(ctx context.Context, ref governance.GovernanceReference) (*governance.GovernanceState, error) {
	return &governance.GovernanceState{
		Reference: ref,
		Status:    governance.GovernanceStatusReady,
	}, nil
}

func TestTaskViewWithGovernanceTimestamp(t *testing.T) {
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	fixedTime := time.Date(2026, 9, 11, 10, 42, 0, 0, time.UTC)
	gov := &fixedTimeReader{refreshedAt: fixedTime}
	server := NewServer(":0", db, gov)

	server.templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	err = server.ProjectRepo.Create(ctx, &domain.Project{
		ID:   "proj-ts",
		Name: "Timestamp Project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	govRef := `{"provider":"solvent","reference_id":"scenario-ts"}`
	err = server.TaskRepo.Create(ctx, &domain.Task{
		ID:            "task-ts",
		ProjectID:     "proj-ts",
		Title:         "Timestamp Task",
		GovernanceRef: &govRef,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/task/task-ts", nil)
	w := httptest.NewRecorder()
	server.handleTask(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	expected := "As of 2026-09-11 10:42 UTC"
	if !strings.Contains(body, expected) {
		t.Errorf("expected %q in rendered body, got: %s", expected, body)
	}
}

func TestTaskViewWithoutGovernanceTimestamp(t *testing.T) {
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &zeroTimeReader{}
	server := NewServer(":0", db, gov)

	server.templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	err = server.ProjectRepo.Create(ctx, &domain.Project{
		ID:   "proj-nots",
		Name: "No Timestamp Project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	govRef := `{"provider":"solvent","reference_id":"scenario-nots"}`
	err = server.TaskRepo.Create(ctx, &domain.Task{
		ID:            "task-nots",
		ProjectID:     "proj-nots",
		Title:         "No Timestamp Task",
		GovernanceRef: &govRef,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/task/task-nots", nil)
	w := httptest.NewRecorder()
	server.handleTask(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "As of") {
		t.Errorf("expected 'As of' timestamp to be absent for zero RefreshedAt, but found it in body: %s", body)
	}
}
