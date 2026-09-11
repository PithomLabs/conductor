package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/PithomLabs/conductor/internal/governance"
)

// GetGovernance handles GET /v1/tasks/{id}/governance
// This is read-only governance observation
func (s *Server) GetGovernance(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskIDFromPath(r.URL.Path)
	if taskID == "" {
		http.Error(w, "missing task ID", http.StatusBadRequest)
		return
	}

	// Get task to check governance_ref
	task, err := s.TaskRepo.GetByID(r.Context(), taskID)
	if err != nil {
		if err.Error() == "record not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If no governance reference, return empty state
	if task.GovernanceRef == nil || *task.GovernanceRef == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "none",
		})
		return
	}

	// Parse governance reference
	var ref governance.GovernanceReference
	if err := json.Unmarshal([]byte(*task.GovernanceRef), &ref); err != nil {
		// If it's not JSON, create a generic reference
		ref = governance.GovernanceReference{
			Provider:    "unknown",
			ReferenceID: *task.GovernanceRef,
		}
	}

	// Read governance state through reader
	state, err := s.Governance.GetState(r.Context(), ref)
	if err != nil {
		// Return what we have without governance state
		state = &governance.GovernanceState{
			Reference:   ref,
			Status:      governance.GovernanceStatusUnknown,
			RefreshedAt: time.Now(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}
