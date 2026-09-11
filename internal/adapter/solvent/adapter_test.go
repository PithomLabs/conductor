package solvent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PithomLabs/conductor/internal/governance"
)

func TestAdapterGetStateWithBeliefRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/beliefs/belief-1/explain" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"belief_id": "belief-1",
				"claim": "test claim",
				"status": "entered",
				"is_promoted": false,
				"is_retracted": false,
				"can_promote": true,
				"remaining_debt": [],
				"human_summary": "test"
			}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	adapter := NewAdapter(client)

	ref := governance.GovernanceReference{
		Provider:    "solvent",
		ReferenceID: "scenario-1",
		Metadata: map[string]interface{}{
			"belief_id": "belief-1",
		},
	}

	state, err := adapter.GetState(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Reference.Provider != "solvent" {
		t.Errorf("expected provider solvent, got %s", state.Reference.Provider)
	}

	if state.Status != governance.GovernanceStatusPending {
		t.Errorf("expected status pending, got %s", state.Status)
	}
}

func TestAdapterGetStateMissingBeliefID(t *testing.T) {
	client := NewClient("http://localhost", "test-key")
	adapter := NewAdapter(client)

	ref := governance.GovernanceReference{
		Provider:    "solvent",
		ReferenceID: "scenario-1",
		Metadata:    map[string]interface{}{},
	}

	_, err := adapter.GetState(context.Background(), ref)
	if err == nil {
		t.Error("expected error for missing belief_id")
	}
}

func TestAdapterGetStateEmptyBeliefID(t *testing.T) {
	client := NewClient("http://localhost", "test-key")
	adapter := NewAdapter(client)

	ref := governance.GovernanceReference{
		Provider:    "solvent",
		ReferenceID: "scenario-1",
		Metadata: map[string]interface{}{
			"belief_id": "",
		},
	}

	_, err := adapter.GetState(context.Background(), ref)
	if err == nil {
		t.Error("expected error for empty belief_id")
	}
}

func TestAdapterGetStateMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	adapter := NewAdapter(client)

	ref := governance.GovernanceReference{
		Provider:    "solvent",
		ReferenceID: "scenario-1",
		Metadata: map[string]interface{}{
			"belief_id": "belief-1",
		},
	}

	_, err := adapter.GetState(context.Background(), ref)
	if err == nil {
		t.Error("expected error for malformed response")
	}
}

func TestAdapterGetStateSolventUnavailable(t *testing.T) {
	client := NewClient("http://localhost:99999", "test-key")
	adapter := NewAdapter(client)

	ref := governance.GovernanceReference{
		Provider:    "solvent",
		ReferenceID: "scenario-1",
		Metadata: map[string]interface{}{
			"belief_id": "belief-1",
		},
	}

	_, err := adapter.GetState(context.Background(), ref)
	if err == nil {
		t.Error("expected error for unavailable Solvent")
	}
}

func TestAdapterGetBeliefState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/beliefs/belief-1/explain" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"belief_id": "belief-1",
				"claim": "test claim",
				"status": "entered",
				"is_promoted": false,
				"is_retracted": false,
				"can_promote": true,
				"remaining_debt": [],
				"human_summary": "test"
			}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	adapter := NewAdapter(client)

	state, err := adapter.GetBeliefState(context.Background(), "scenario-1", "belief-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Reference.Provider != "solvent" {
		t.Errorf("expected provider solvent, got %s", state.Reference.Provider)
	}

	if state.Status != governance.GovernanceStatusPending {
		t.Errorf("expected status pending, got %s", state.Status)
	}
}

func TestAdapterGetBeliefStateRetracted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"belief_id": "belief-1",
			"claim": "test claim",
			"status": "retracted",
			"is_promoted": false,
			"is_retracted": true,
			"can_promote": false,
			"remaining_debt": [],
			"human_summary": "test"
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	adapter := NewAdapter(client)

	state, err := adapter.GetBeliefState(context.Background(), "scenario-1", "belief-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Status != governance.GovernanceStatusBlocked {
		t.Errorf("expected status blocked, got %s", state.Status)
	}

	if len(state.Blockers) == 0 || state.Blockers[0] != "belief retracted" {
		t.Errorf("expected blocker 'belief retracted', got %v", state.Blockers)
	}
}


func TestAdapterGetStateRequiresReferenceID(t *testing.T) {
	client := NewClient("http://localhost", "test-key")
	adapter := NewAdapter(client)

	ref := governance.GovernanceReference{
		Provider: "solvent",
	}

	_, err := adapter.GetState(context.Background(), ref)
	if err == nil {
		t.Error("expected error for missing reference_id")
	}
}
