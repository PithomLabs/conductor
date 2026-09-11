package web

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

// Server is the HTTP server for Conductor UI
type Server struct {
	Addr         string
	TaskRepo     *store.TaskRepository
	ProjectRepo  *store.ProjectRepository
	ActivityRepo *store.ActivityRepository
	Governance   governance.GovernanceReader
	templates    *template.Template
}

// NewServer creates a new web server
func NewServer(addr string, db *store.DB, gov governance.GovernanceReader) *Server {
	return &Server{
		Addr:         addr,
		TaskRepo:     store.NewTaskRepository(db),
		ProjectRepo:  store.NewProjectRepository(db),
		ActivityRepo: store.NewActivityRepository(db),
		Governance:   gov,
	}
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe() error {
	var err error
	s.templates, err = template.ParseGlob("internal/web/templates/*.html")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/project/", s.handleProject)
	mux.HandleFunc("/task/", s.handleTask)

	log.Printf("Conductor UI listening on %s", s.Addr)
	return http.ListenAndServe(s.Addr, mux)
}

// renderTemplate renders a template with data
func (s *Server) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleDashboard handles the main dashboard
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	projects, err := s.ProjectRepo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.renderTemplate(w, "layout.html", map[string]interface{}{
		"Title":    "Conductor Dashboard",
		"Projects": projects,
	})
}

// handleProject handles project detail view
func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	projectID := extractProjectID(r.URL.Path)
	if projectID == "" {
		http.NotFound(w, r)
		return
	}

	project, err := s.ProjectRepo.GetByID(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	tasks, err := s.TaskRepo.ListByProject(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Group tasks by status for Kanban view
	taskBoard := map[string][]*domain.Task{
		"proposed": {},
		"active":   {},
		"review":   {},
		"accepted": {},
		"blocked":  {},
	}
	for _, task := range tasks {
		taskBoard[task.Status] = append(taskBoard[task.Status], task)
	}

	data := map[string]interface{}{
		"Title":     project.Name,
		"Project":   project,
		"TaskBoard": taskBoard,
	}
	s.renderTemplate(w, "layout.html", data)
}

// handleTask handles task detail view
func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskID(r.URL.Path)
	if taskID == "" {
		http.NotFound(w, r)
		return
	}

	task, err := s.TaskRepo.GetByID(r.Context(), taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	activities, err := s.ActivityRepo.ListByTask(r.Context(), taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get governance state if task has governance_ref
	var governanceState *governance.GovernanceState
	if task.GovernanceRef != nil && *task.GovernanceRef != "" {
		var ref governance.GovernanceReference
		if err := json.Unmarshal([]byte(*task.GovernanceRef), &ref); err == nil {
			governanceState, _ = s.Governance.GetState(r.Context(), ref)
		}
	}

	data := map[string]interface{}{
		"Title":           task.Title,
		"Task":            task,
		"Activities":      activities,
		"GovernanceState": governanceState,
	}
	s.renderTemplate(w, "layout.html", data)
}

// extractProjectID extracts project ID from path
func extractProjectID(path string) string {
	// /project/{id}
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// extractTaskID extracts task ID from path
func extractTaskID(path string) string {
	// /task/{id}
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
