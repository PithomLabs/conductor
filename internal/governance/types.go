// Package governance defines the read-only governance state types.
//
// V1 Resource Access Model (explicit scope):
//
// Conductor v1 is a single trusted coordination workspace.
//
// Authentication establishes caller identity (WHO is calling Conductor),
// not per-project authorization. Resource IDs (UUIDs) reduce enumeration
// risk but are NOT authorization boundaries. An authenticated caller has
// access to the instance's projects and tasks. Cross-project isolation
// is explicitly out of scope for v1.
//
// This package provides read-only views of external governance state.
// Conductor does not broker execution or enforce authorization.
// CheckAuthorization is a reserved governance capability, not surfaced
// in HTTP/MCP for v1.

package governance

import "time"

// GovernanceReference is a reference to external governance context.
// provider is interpreted by Conductor for routing; reference_id and metadata are opaque.
type GovernanceReference struct {
	Provider    string                 `json:"provider"`     // Routing selector: "solvent", "manual", etc.
	ReferenceID string                 `json:"reference_id"` // Opaque to Conductor, interpreted by external system
	Metadata    map[string]interface{} `json:"metadata"`     // Opaque to Conductor, interpreted by external system
}

// GovernanceState represents the current state of external governance.
type GovernanceState struct {
	Reference   GovernanceReference `json:"reference"`
	Status      string              `json:"status"`      // "unknown", "pending", "ready", "blocked"
	Blockers    []string            `json:"blockers"`    // Generic reasons
	RefreshedAt time.Time           `json:"refreshed_at"`
}

// Governance state status constants
const (
	GovernanceStatusUnknown = "unknown"
	GovernanceStatusPending = "pending"
	GovernanceStatusReady   = "ready"
	GovernanceStatusBlocked = "blocked"
)

// AuthorizationResult represents whether an action is permitted.
// This is an INFORMATIONAL QUERY, not a Conductor authorization decision.
type AuthorizationResult struct {
	Allowed   bool     `json:"allowed"`
	Reason    string   `json:"reason"`    // Human-readable, generic
	Providers []string `json:"providers"` // Which systems were consulted
}
