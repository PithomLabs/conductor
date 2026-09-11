package api

import (
	"net/http"
	"strings"
)

// registerRoutes sets up all API routes
func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/", s.dispatch)
}

// dispatch routes all /v1/ requests
func (s *Server) dispatch(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/")
	segments := strings.Split(path, "/")

	// /v1/projects
	if len(segments) == 1 && segments[0] == "projects" {
		switch r.Method {
		case http.MethodGet:
			s.ListProjects(w, r)
		case http.MethodPost:
			s.CreateProject(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /v1/projects/{id}
	if len(segments) == 2 && segments[0] == "projects" {
		switch r.Method {
		case http.MethodGet:
			s.GetProject(w, r)
		case http.MethodPut:
			s.UpdateProject(w, r)
		case http.MethodDelete:
			s.DeleteProject(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /v1/projects/{id}/tasks
	if len(segments) == 3 && segments[0] == "projects" && segments[2] == "tasks" {
		switch r.Method {
		case http.MethodGet:
			s.ListTasks(w, r)
		case http.MethodPost:
			s.CreateTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /v1/tasks/{id}/action
	if len(segments) == 3 && segments[0] == "tasks" {
		action := segments[2]
		switch action {
		case "claim":
			s.ClaimTask(w, r)
		case "submit":
			s.SubmitTask(w, r)
		case "accept":
			s.AcceptTask(w, r)
		case "reject":
			s.RejectTask(w, r)
		case "blocker":
			s.ReportBlocker(w, r)
		case "resolve":
			s.ResolveBlocker(w, r)
		case "activity":
			s.PostActivity(w, r)
		case "release":
			s.ReleaseTask(w, r)
		case "governance":
			s.GetGovernance(w, r)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
		return
	}

	// /v1/tasks/{id}
	if len(segments) == 2 && segments[0] == "tasks" {
		switch r.Method {
		case http.MethodGet:
			s.GetTask(w, r)
		case http.MethodPatch:
			s.UpdateTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}
