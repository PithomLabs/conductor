package service

import (
	"context"
	"testing"

	"github.com/PithomLabs/conductor/internal/governance"
)

func TestGovernanceServiceGetState(t *testing.T) {
	service := NewGovernanceService()

	// Register a null reader
	service.RegisterProvider("null", &governance.NullReader{})

	ref := governance.GovernanceReference{
		Provider:    "null",
		ReferenceID: "test-1",
	}

	state, err := service.GetState(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Status != governance.GovernanceStatusUnknown {
		t.Errorf("expected status unknown, got %s", state.Status)
	}
}

func TestGovernanceServiceUnknownProvider(t *testing.T) {
	service := NewGovernanceService()

	ref := governance.GovernanceReference{
		Provider:    "unknown",
		ReferenceID: "test-1",
	}

	state, err := service.GetState(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Status != governance.GovernanceStatusUnknown {
		t.Errorf("expected status unknown, got %s", state.Status)
	}

	if len(state.Blockers) == 0 {
		t.Error("expected blocker for unknown provider")
	}

	if state.RefreshedAt.IsZero() {
		t.Error("expected RefreshedAt to be set on unknown provider observation")
	}
}
