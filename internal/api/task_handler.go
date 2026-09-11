package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/store"
)

// ListTasks handles GET /v1/projects/{id}/tasks
func (s *Server) ListTasks(w http.ResponseWriter, r *http.Request) {
	projectID := extractIDFromPath(r.URL.Path, "projects")
	if projectID == "" {
		http.Error(w, "missing project ID", http.StatusBadRequest)
		return
	}

	tasks, err := s.TaskRepo.ListByProject(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetTask handles GET /v1/tasks/{id}
func (s *Server) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	task, err := s.TaskRepo.GetByID(r.Context(), taskID)
	if err != nil {
		if err.Error() == "record not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}


// CreateTask handles POST /v1/projects/{id}/tasks
func (s *Server) CreateTask(w http.ResponseWriter, r *http.Request) {
	projectID := extractIDFromPath(r.URL.Path, "projects")
	if projectID == "" {
		http.Error(w, "missing project ID", http.StatusBadRequest)
		return
	}

	// governance_ref may be supplied at creation time. Once set, it is immutable.
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	task.ProjectID = projectID

	// Reject lifecycle/assignment fields — do not silently strip
	if task.Status != "" && task.Status != domain.TaskStatusProposed {
		http.Error(w, "lifecycle bypass: status cannot be set at creation", http.StatusBadRequest)
		return
	}
	if task.CurrentAgent != nil {
		http.Error(w, "lifecycle bypass: current_agent cannot be set at creation", http.StatusBadRequest)
		return
	}

	if err := s.TaskRepo.Create(r.Context(), &task); err != nil {
		if err == store.ErrLifecycleBypass {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// NOTE: Cannot update status or current_agent through this endpoint
func (s *Server) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	var fields store.TaskUpdateFields
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.TaskRepo.Update(r.Context(), taskID, fields); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated task
	task, _ := s.TaskRepo.GetByID(r.Context(), taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// ClaimTask handles POST /v1/tasks/{id}/claim
func (s *Server) ClaimTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	agentID := agentIDFromContext(r.Context())

	err := s.TaskRepo.Claim(r.Context(), taskID, agentID)
	if err != nil {
		if err == store.ErrReleaseFailed {
			http.Error(w, "release only permitted from active state", http.StatusConflict)
			return
		}
		if err == store.ErrClaimFailed {
			http.Error(w, "task already claimed", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated task
	task, _ := s.TaskRepo.GetByID(r.Context(), taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// SubmitTask handles POST /v1/tasks/{id}/submit
func (s *Server) SubmitTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	agentID := agentIDFromContext(r.Context())

	err := s.TaskRepo.Transition(r.Context(), taskID,
		domain.TaskStatusActive, domain.TaskStatusReview,
		"agent", agentID, "task.submitted", "{}")
	if err != nil {
		if err == store.ErrInvalidTransition {
			http.Error(w, "invalid transition", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated task
	task, _ := s.TaskRepo.GetByID(r.Context(), taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// AcceptTask handles POST /v1/tasks/{id}/accept
func (s *Server) AcceptTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	agentID := agentIDFromContext(r.Context())

	err := s.TaskRepo.Transition(r.Context(), taskID,
		domain.TaskStatusReview, domain.TaskStatusAccepted,
		"agent", agentID, "task.accepted", "{}")
	if err != nil {
		if err == store.ErrInvalidTransition {
			http.Error(w, "invalid transition", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated task
	task, _ := s.TaskRepo.GetByID(r.Context(), taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// RejectTask handles POST /v1/tasks/{id}/reject
func (s *Server) RejectTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	agentID := agentIDFromContext(r.Context())

	err := s.TaskRepo.Transition(r.Context(), taskID,
		domain.TaskStatusReview, domain.TaskStatusActive,
		"agent", agentID, "task.rejected", "{}")
	if err != nil {
		if err == store.ErrInvalidTransition {
			http.Error(w, "invalid transition", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated task
	task, _ := s.TaskRepo.GetByID(r.Context(), taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// ReportBlocker handles POST /v1/tasks/{id}/blocker
func (s *Server) ReportBlocker(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	agentID := agentIDFromContext(r.Context())

	err := s.TaskRepo.Transition(r.Context(), taskID,
		domain.TaskStatusActive, domain.TaskStatusBlocked,
		"agent", agentID, "task.blocked", "{}")
	if err != nil {
		if err == store.ErrInvalidTransition {
			http.Error(w, "invalid transition", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated task
	task, _ := s.TaskRepo.GetByID(r.Context(), taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// ResolveBlocker handles POST /v1/tasks/{id}/resolve
func (s *Server) ResolveBlocker(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	agentID := agentIDFromContext(r.Context())

	err := s.TaskRepo.Transition(r.Context(), taskID,
		domain.TaskStatusBlocked, domain.TaskStatusActive,
		"agent", agentID, "task.unblocked", "{}")
	if err != nil {
		if err == store.ErrInvalidTransition {
			http.Error(w, "invalid transition", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated task
	task, _ := s.TaskRepo.GetByID(r.Context(), taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// PostActivity handles POST /v1/tasks/{id}/activity
func (s *Server) PostActivity(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	agentID := agentIDFromContext(r.Context())

	var activity domain.Activity
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	activity.TaskID = taskID
	activity.ActorType = "agent"
	activity.ActorID = agentID

	if err := s.ActivityRepo.Append(r.Context(), &activity); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(activity)
}

// extractTaskIDFromPath extracts task ID from /v1/tasks/{id} or /v1/tasks/{id}/action
func extractTaskIDFromPath(path string) string {
	segments := strings.Split(strings.TrimPrefix(path, "/v1/"), "/")
	// segments[0] should be "tasks", segments[1] should be the ID
	if len(segments) >= 2 && segments[0] == "tasks" {
		return segments[1]
	}
	return ""
}

// ReleaseTask handles POST /v1/tasks/{id}/release
func (s *Server) ReleaseTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	agentID := agentIDFromContext(r.Context())

	err := s.TaskRepo.Release(r.Context(), taskID, agentID)
	if err != nil {
		if err == store.ErrReleaseFailed {
			http.Error(w, "release only permitted from active state", http.StatusConflict)
			return
		}
		if err == store.ErrClaimFailed {
			http.Error(w, "not assigned to this task or task not active", http.StatusConflict)
			return
		}
		if err == store.ErrNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	task, _ := s.TaskRepo.GetByID(r.Context(), taskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
