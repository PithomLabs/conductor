package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

var testCredentials = map[string]string{
	"test-key-1": "agent-1",
	"test-key-2": "agent-2",
}

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := &governance.NullReader{}
	return NewServer(":0", db, gov, testCredentials)
}

// wrappedMux returns an auth-wrapped mux with routes registered
func wrappedMux(server *Server) *http.ServeMux {
	authMux := http.NewServeMux()
	server.registerRoutes(authMux)
	wrapped := http.NewServeMux()
	wrapped.Handle("/v1/", NewAuthMiddleware(server.Credentials)(authMux))
	return wrapped
}

func TestAPIProjectCRUD(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	body := `{"id":"proj-1","name":"Test Project","description":"A test project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/projects/proj-1", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var project map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if project["id"] != "proj-1" {
		t.Errorf("expected id proj-1, got %v", project["id"])
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/v1/projects/proj-1", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPITaskLifecycle(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	body := `{"id":"proj-1","name":"Test Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body = `{"id":"task-1","title":"Test Task"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/tasks/task-1/claim", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/tasks/task-1/submit", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/tasks/task-1/accept", nil)
	req.Header.Set("X-API-Key", "test-key-2")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/tasks/task-1", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var task map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if task["status"] != "accepted" {
		t.Errorf("expected status accepted, got %v", task["status"])
	}
}

func TestAPILifecycleBypassPrevention(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	body := `{"id":"proj-1","name":"Test Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body = `{"id":"task-1","title":"Test Task"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Try to update title (lifecycle fields not in TaskUpdateFields, so ignored)
	body = `{"title":"Updated Title"}`
	req = httptest.NewRequest(http.MethodPatch, "/v1/tasks/task-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodGet, "/v1/tasks/task-1", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var task map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if task["title"] != "Updated Title" {
		t.Errorf("expected title Updated Title, got %v", task["title"])
	}
	if task["status"] != "proposed" {
		t.Errorf("expected status proposed, got %v", task["status"])
	}
}

func TestAPIUnauthenticatedRequestRejected(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	body := `{"id":"proj-1","name":"Test Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body = `{"id":"task-1","title":"Test Task"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// No API key — should fail
	req = httptest.NewRequest(http.MethodPost, "/v1/tasks/task-1/claim", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", w.Code)
	}
}

func TestAPIInvalidCredentialRejected(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	body := `{"id":"proj-1","name":"Test Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body = `{"id":"task-1","title":"Test Task"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodPost, "/v1/tasks/task-1/claim", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid credential, got %d", w.Code)
	}
}

func TestAPIAuthenticatedRequestDerivesIdentity(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	body := `{"id":"proj-1","name":"Test Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body = `{"id":"task-1","title":"Test Task"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Claim with test-key-1 -> agent-1
	req = httptest.NewRequest(http.MethodPost, "/v1/tasks/task-1/claim", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/tasks/task-1", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var task map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if task["current_agent"] != "agent-1" {
		t.Errorf("expected current_agent 'agent-1', got %v", task["current_agent"])
	}
}

func TestAPIInvalidTransition(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	body := `{"id":"proj-1","name":"Test Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body = `{"id":"task-1","title":"Test Task"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Try to submit without claiming first (invalid transition: proposed -> review)
	req = httptest.NewRequest(http.MethodPost, "/v1/tasks/task-1/submit", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid transition, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTPCreateTaskWithGovernanceRef(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	// Create project
	body := `{"id":"proj-gov","name":"Gov Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d", w.Code)
	}

	// Create task with governance_ref
	body = `{"id":"task-gov","title":"Governed Task","governance_ref":"{\"provider\":\"solvent\",\"reference_id\":\"scenario-1\"}"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-gov/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create task: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// GET the task and verify governance_ref persisted
	req = httptest.NewRequest(http.MethodGet, "/v1/tasks/task-gov", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var task map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if task["governance_ref"] == nil {
		t.Error("expected governance_ref to be set")
	}
}

func TestHTTPCreateTaskWithoutGovernanceRef(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	body := `{"id":"proj-nogov","name":"NoGov Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d", w.Code)
	}

	body = `{"id":"task-nogov","title":"Ungoverned Task"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-nogov/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create task: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/tasks/task-nogov", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var task map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if task["governance_ref"] != nil {
		t.Errorf("expected governance_ref nil, got %v", task["governance_ref"])
	}
}

func TestHTTPUpdateTaskDoesNotAlterGovernanceRef(t *testing.T) {
	server := setupTestServer(t)
	mux := wrappedMux(server)

	// Create project
	body := `{"id":"proj-patch","name":"Patch Project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d", w.Code)
	}

	// Create task with governance_ref
	govRefBody := map[string]interface{}{
		"id":             "task-patch",
		"title":          "Patch Test",
		"governance_ref": `{"provider":"solvent","reference_id":"scenario-4"}`,
	}
	govRefJSON, _ := json.Marshal(govRefBody)
	body = string(govRefJSON)
	req = httptest.NewRequest(http.MethodPost, "/v1/projects/proj-patch/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create task: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// PATCH title
	patchBody := `{"title":"Patched Title"}`
	req = httptest.NewRequest(http.MethodPatch, "/v1/tasks/task-patch", bytes.NewBufferString(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("patch task: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify governance_ref unchanged
	req = httptest.NewRequest(http.MethodGet, "/v1/tasks/task-patch", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var task map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if task["title"] != "Patched Title" {
		t.Errorf("expected title 'Patched Title', got %v", task["title"])
	}
	if task["governance_ref"] == nil {
		t.Error("expected governance_ref preserved after PATCH")
	}
}
