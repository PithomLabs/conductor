package governance

import "context"

// GovernanceReader is the internal read-only interface for querying
// external governance systems. This is strictly a read-side abstraction.
// Conductor uses this to observe governance state; it never brokers execution.
//
// V1 scope: Conductor observes governance state only.
// Authorization queries go directly Agent → Solvent, not through Conductor.
// CheckAuthorization is intentionally absent from this interface.
type GovernanceReader interface {
	// GetState returns the current governance state for a reference.
	GetState(ctx context.Context, ref GovernanceReference) (*GovernanceState, error)
}
