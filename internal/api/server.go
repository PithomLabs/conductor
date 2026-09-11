package api

import (
	"log"
	"net/http"

	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

// Server is the HTTP server for Conductor API
type Server struct {
	Addr           string
	TaskRepo       *store.TaskRepository
	ProjectRepo    *store.ProjectRepository
	ActivityRepo   *store.ActivityRepository
	DependencyRepo *store.DependencyRepository
	Governance     governance.GovernanceReader
	Credentials    map[string]string // API key -> stable actor ID
}

// NewServer creates a new API server
func NewServer(addr string, db *store.DB, gov governance.GovernanceReader, credentials map[string]string) *Server {
	return &Server{
		Addr:           addr,
		TaskRepo:       store.NewTaskRepository(db),
		ProjectRepo:    store.NewProjectRepository(db),
		ActivityRepo:   store.NewActivityRepository(db),
		DependencyRepo: store.NewDependencyRepository(db),
		Governance:     gov,
		Credentials:    credentials,
	}
}

// ListenAndServe starts the HTTP server with authentication
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Wrap all /v1/ routes with authentication
	authMux := http.NewServeMux()
	authMux.Handle("/v1/", NewAuthMiddleware(s.Credentials)(mux))

	log.Printf("Conductor API listening on %s", s.Addr)
	return http.ListenAndServe(s.Addr, authMux)
}
