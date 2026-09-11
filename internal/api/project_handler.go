package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PithomLabs/conductor/internal/domain"
)

// ListProjects handles GET /v1/projects
func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.ProjectRepo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// GetProject handles GET /v1/projects/{id}
func (s *Server) GetProject(w http.ResponseWriter, r *http.Request) {
	projectID := extractIDFromPath(r.URL.Path, "projects")
	if projectID == "" {
		http.Error(w, "missing project ID", http.StatusBadRequest)
		return
	}

	project, err := s.ProjectRepo.GetByID(r.Context(), projectID)
	if err != nil {
		if err.Error() == "record not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// CreateProject handles POST /v1/projects
func (s *Server) CreateProject(w http.ResponseWriter, r *http.Request) {
	var project domain.Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.ProjectRepo.Create(r.Context(), &project); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

// UpdateProject handles PUT /v1/projects/{id}
func (s *Server) UpdateProject(w http.ResponseWriter, r *http.Request) {
	projectID := extractIDFromPath(r.URL.Path, "projects")
	if projectID == "" {
		http.Error(w, "missing project ID", http.StatusBadRequest)
		return
	}

	var project domain.Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	project.ID = projectID

	if err := s.ProjectRepo.Update(r.Context(), &project); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// DeleteProject handles DELETE /v1/projects/{id}
func (s *Server) DeleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := extractIDFromPath(r.URL.Path, "projects")
	if projectID == "" {
		http.Error(w, "missing project ID", http.StatusBadRequest)
		return
	}

	if err := s.ProjectRepo.Delete(r.Context(), projectID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// extractIDFromPath extracts an ID after the given prefix segment
// e.g., extractIDFromPath("/v1/projects/proj-1/tasks", "projects") => "proj-1"
func extractIDFromPath(path, after string) string {
	segments := strings.Split(strings.TrimPrefix(path, "/v1/"), "/")
	for i, seg := range segments {
		if seg == after && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return ""
}
