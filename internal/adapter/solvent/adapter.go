package solvent

import (
	"context"
	"fmt"

	"github.com/PithomLabs/conductor/internal/governance"
)

// Adapter implements GovernanceReader for Solvent.
// This is a READ-ONLY adapter. It queries Solvent state;
// it does not broker execution.
type Adapter struct {
	client *Client
}

// NewAdapter creates a new Solvent adapter.
func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

// GetState returns the governance state for a reference.
// The reference metadata must contain "belief_id" for Solvent-specific routing.
// Missing belief_id returns an explicit error, not "unknown".
func (a *Adapter) GetState(ctx context.Context, ref governance.GovernanceReference) (*governance.GovernanceState, error) {
	if ref.ReferenceID == "" {
		return nil, fmt.Errorf("reference_id is required")
	}

	beliefID, ok := ref.Metadata["belief_id"]
	if !ok || beliefID == "" {
		return nil, fmt.Errorf("solvent adapter: belief_id required in governance reference metadata")
	}

	beliefIDStr, ok := beliefID.(string)
	if !ok || beliefIDStr == "" {
		return nil, fmt.Errorf("solvent adapter: belief_id must be a non-empty string")
	}

	explain, err := a.client.GetBeliefExplain(ctx, ref.ReferenceID, beliefIDStr)
	if err != nil {
		return nil, fmt.Errorf("get belief explain: %w", err)
	}

	return translateBeliefExplain(ref, explain), nil
}

// GetBeliefState returns governance state for a specific belief reference.
// This is the reference-specific query that uses authoritative Solvent state.
func (a *Adapter) GetBeliefState(ctx context.Context, scenarioID, beliefID string) (*governance.GovernanceState, error) {
	ref := governance.GovernanceReference{
		Provider:    "solvent",
		ReferenceID: scenarioID,
		Metadata: map[string]interface{}{
			"belief_id": beliefID,
		},
	}

	explain, err := a.client.GetBeliefExplain(ctx, scenarioID, beliefID)
	if err != nil {
		return nil, fmt.Errorf("get belief explain: %w", err)
	}

	return translateBeliefExplain(ref, explain), nil
}
