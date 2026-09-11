package solvent

import (
	"time"

	"github.com/PithomLabs/conductor/internal/governance"
)

// translateBeliefExplain translates Solvent belief explain to generic governance state.
// This is reference-specific translation, NOT aggregate statistics.
func translateBeliefExplain(ref governance.GovernanceReference, explain *BeliefExplainResponse) *governance.GovernanceState {
	status := "unknown"
	blockers := []string{}

	// Use authoritative Solvent state for this specific reference
	switch {
	case explain.IsRetracted:
		status = governance.GovernanceStatusBlocked
		blockers = append(blockers, "belief retracted")
	case explain.IsPromoted:
		status = governance.GovernanceStatusReady
	case explain.CanPromote:
		status = governance.GovernanceStatusPending
	default:
		status = governance.GovernanceStatusBlocked
		if explain.PromotionBlockedReason != "" {
			blockers = append(blockers, explain.PromotionBlockedReason)
		}
	}

	return &governance.GovernanceState{
		Reference:   ref,
		Status:      status,
		Blockers:    blockers,
		RefreshedAt: time.Now(),
	}
}

// translateAuthorizationResult translates Solvent authorization result to generic authorization result.
// This is an informational query, NOT Conductor authorization.
func translateAuthorizationResult(ref governance.GovernanceReference, action string, result *AuthResult) *governance.AuthorizationResult {
	return &governance.AuthorizationResult{
		Allowed:   result.Allowed,
		Reason:    result.Reason,
		Providers: []string{"solvent"},
	}
}
