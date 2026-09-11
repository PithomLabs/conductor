package governance

import (
	"context"
	"time"
)

// NullReader is the default GovernanceReader when no external governance is configured.
type NullReader struct{}

func (r *NullReader) GetState(ctx context.Context, ref GovernanceReference) (*GovernanceState, error) {
	return &GovernanceState{
		Reference:   ref,
		Status:      GovernanceStatusUnknown,
		Blockers:    []string{"no external governance system configured"},
		RefreshedAt: time.Now(),
	}, nil
}
