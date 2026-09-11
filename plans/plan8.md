# CONDUCTOR — FINAL TARGETED REMEDIATION

## Source

Second adversarial review. Verdict: **YELLOW** — one genuine critical regression (C1), operational hardening (C2), consistency bug (H2), attribution bug (M2), dead code (M1). Architecture is substantially stronger than first review.

## Locked Architecture

```
Agent      = Agency
Conductor  = Coordination
Solvent    = Authority
Executor   = Effect

CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION
```

No Solvent changes. No RBAC. No multi-tenancy. No capability matching. No execution features. No project membership. No per-project authorization.

---

## 1. C1 — Fix Release Lifecycle Bypass (CRITICAL)

### Problem

`Release()` in `internal/store/task_repo.go:277-313` checks `currentAgent` ownership but does NOT check `currentStatus == active`. A task in `review`, `blocked`, or `accepted` state can be released to `proposed` if the caller is the assigned agent.

This directly violates the Conductor lifecycle invariant. Release is valid ONLY from `active → proposed`.

### Current Code (defective)

```go
func (r *TaskRepository) Release(ctx context.Context, taskID string, agentID string) error {
    return r.db.Transaction(func(tx *sql.Tx) error {
        var currentAgent *string
        var currentStatus string
        err := tx.QueryRowContext(ctx,
            "SELECT current_agent, status FROM conductor_task WHERE id = ?", taskID).Scan(
            &currentAgent, &currentStatus)
        // ... nil check on currentAgent ...
        // BUG: currentStatus is read but never checked
        _, err = tx.ExecContext(ctx,
            "UPDATE conductor_task SET current_agent = NULL, status = 'proposed' ...",
            taskID)
```

### Fix

Replace read-then-update with a single atomic SQL operation:

```sql
UPDATE conductor_task
SET current_agent = NULL,
    status = 'proposed',
    updated_at = datetime('now')
WHERE id = ?
  AND current_agent = ?
  AND status = 'active'
```

Require `RowsAffected == 1`. Return `ErrReleaseFailed` if 0 rows affected.

This makes release atomic with its precondition check — no TOCTOU gap.

### Files

| File | Change |
|------|--------|
| `internal/store/sqlite.go` | Add `var ErrReleaseFailed` |
| `internal/store/task_repo.go` | Rewrite `Release()` to atomic SQL with status check |
| `internal/api/task_handler.go` | `ReleaseTask` handler: map `ErrReleaseFailed` → HTTP 409 |
| `internal/mcp/tools.go` | No change — error propagates as string |
| `internal/store/store_test.go` | Add 5 new tests (see below) |

### Tests to Add

| Test | Setup | Expect |
|------|-------|--------|
| `TestReleaseFromReviewRejected` | claim → submit (active→review) → release | `ErrReleaseFailed` |
| `TestReleaseFromBlockedRejected` | claim → report blocker (active→blocked) → release | `ErrReleaseFailed` |
| `TestReleaseFromAcceptedRejected` | claim → submit → accept (review→accepted) → release | `ErrReleaseFailed` |
| `TestReleaseFromCancelledRejected` | create → cancel (proposed→cancelled) → release | `ErrReleaseFailed` |
| `TestReleaseFromProposedRejected` | create (proposed, unassigned) → release by any agent | `ErrReleaseFailed` |

### Tests That Continue to Pass (no changes)

- `TestReleaseByAssignedAgent` — active → proposed ✓
- `TestReleaseByOtherAgentRejected` — wrong agent → error ✓
- `TestReleaseActivityRecorded` — activity logged ✓

---

## 2. C2 — Web UI Trust Boundary

### Problem

`cmd/conductor/main.go` passes `*addr` (default `:8080`) to the web server, which binds all network interfaces. For a POC, the web UI should default to loopback.

### Fix

**`cmd/conductor/main.go`** — In the `case "web":` branch:
- If `*addr` was not explicitly overridden by the user, default to `127.0.0.1:8080`
- Detection: check if `*addr == ":8080"` (the flag default), then override to `127.0.0.1:8080`
- If user explicitly passes `--addr 0.0.0.0:8080`, respect it

**`internal/web/web_test.go`** — Add `TestWebDefaultsToLocalBinding`:
- Verify that when web mode is constructed without explicit addr, the server address contains `127.0.0.1`

### Documentation

Add comment in `main.go` and `web/server.go`:

```
// Default web binding is loopback (127.0.0.1) for trusted-local POC.
// Remote exposure requires an authenticated deployment boundary in the future.
```

---

## 3. H2 — Governance Error Consistency

### Problem

HTTP `GetGovernance` (`governance_handler.go:50-57`) catches `GetState` errors and returns a `GovernanceState{status: "unknown"}` — correct behavior.

MCP `getGovernance` (`tools.go:305-308`) propagates the raw error as `MCPResponse{Error: err.Error()}` — inconsistent with HTTP.

Same governance failure produces different shapes depending on interface.

### Fix

**`internal/mcp/tools.go`** — `getGovernance` handler:

```go
state, err := s.governance.GetState(ctx, ref)
if err != nil {
    state = &governance.GovernanceState{
        Reference: ref,
        Status:    governance.GovernanceStatusUnknown,
        Blockers:  []string{"provider unavailable"},
    }
}
return MCPResponse{Result: state}
```

This aligns with the HTTP handler: provider failure → `unknown` status, not error.

### UI Distinction

The web UI template already shows governance state. When `status == "unknown"` and `blockers` contains "provider unavailable" or "no external governance system configured", this is distinguishable from governance being absent (`status == "none"`).

No template changes needed — the existing template renders whatever state is returned.

### Tests

| Test | Setup | Expect |
|------|-------|--------|
| `TestMCPGetGovernanceProviderFailure` | Governance reader returns error | `GovernanceState{status: "unknown", blockers: ["provider unavailable"]}` |
| `TestHTTPGetGovernanceProviderFailure` | Same reader, HTTP handler | Same `GovernanceState` shape |

Both should return identical `GovernanceState` JSON for the same failure.

---

## 4. M2 — Deterministic MCP Identity

### Problem

`main.go:67-71` iterates a Go map to pick the first credential's actor ID:

```go
agentID := "agent-mcp"
for _, id := range credentials {
    agentID = id
    break
}
```

Go map iteration order is randomized. Different runs may produce different MCP agent IDs. This breaks attribution.

### Fix

**`cmd/conductor/main.go`** — MCP identity:

```go
case "mcp":
    agentID := os.Getenv("CONDUCTOR_MCP_AGENT_ID")
    if agentID == "" {
        agentID = "agent-mcp"
    }
    server := mcp.NewServer(db, govService, agentID)
```

Remove the credential-map iteration entirely for MCP mode. Identity comes from explicit env var or deterministic default.

### Why Not Use Credential Map

The MCP agent identity is about *which agent is using the MCP interface*, not about *which API key was used to authenticate*. These are different concepts. The MCP agent ID should be explicitly configured.

### Documentation

Add comment in `main.go`:

```go
// MCP agent identity is deterministic: explicit env var or fixed default.
// Do NOT derive from credential map iteration (non-deterministic).
```

---

## 5. M1 — Remove Dead `compareKeys`

### Problem

`auth.go:57-61` exports `compareKeys` which is never called. The middleware uses map lookup (`credentials[apiKey]` at line 26). Dead exported code creates misleading security impression.

### Fix

**`internal/api/auth.go`**:
- Remove `compareKeys` function (lines 57-61)
- Remove `crypto/subtle` import (line 5)

### Tests

No new tests needed. Existing `api_test.go` tests verify auth works correctly without `compareKeys`.

---

## 6. L2 — Governance Ref Envelope Validation

### Status

**Already satisfied. No code changes needed.**

Current behavior:
- `task_repo.go:Create()` stores `governance_ref` as opaque JSON string
- `governance_handler.go:41` and `tools.go:298` parse only the generic envelope: `provider`, `reference_id`, `metadata`
- Solvent-specific `belief_id` is handled by the Solvent adapter, not Conductor core
- No Solvent-specific validation in Conductor code

### Verification

Confirm by code inspection:
- `GovernanceReference` struct has `Provider`, `ReferenceID`, `Metadata` — all generic
- Adapter reads `Metadata["belief_id"]` — adapter responsibility, not Conductor
- No Conductor code validates Solvent-specific fields

---

## 7. H3 — V1 Access Model

### Status

**Already satisfied. No code changes needed.**

Current behavior:
- Single trusted workspace
- `CONDUCTOR_API_KEY` authenticates identity, not authorization
- No project membership, no RBAC, no per-project authorization
- UUIDs reduce enumeration, not authorization
- `governance/types.go` doc comments document this explicitly

### Verification

Confirm by code inspection:
- No project membership tables
- No per-project access checks
- Auth middleware (`auth.go:26`) does simple map lookup, no project scoping
- Documentation in `governance/types.go:1-16` and `governance/reader.go:1-11` states V1 scope

---

## Execution Order

| Step | Finding | Priority | Risk |
|------|---------|----------|------|
| 1 | C1 — Release lifecycle bypass | CRITICAL | Must fix — regression from prior remediation |
| 2 | M1 — Remove dead compareKeys | LOW | Trivial deletion |
| 3 | M2 — Deterministic MCP identity | MEDIUM | Simple env var |
| 4 | C2 — Web local binding | MEDIUM | Default address change |
| 5 | H2 — Governance error consistency | MEDIUM | MCP error → state |
| 6 | L2, H3 — Verify | LOW | Code inspection only |
| 7 | Full test suite | — | — |
| 8 | Verification checklist | — | — |
| 9 | Report | — | — |

---

## Verification Checklist

After all changes, verify:

1. **No lifecycle bypass** — `Release()` only succeeds from `active → proposed`
2. **No authentication bypass on HTTP** — all `/v1/` routes require valid API key
3. **MCP trusted-local boundary explicitly documented** — `CONDUCTOR_MCP_AGENT_ID` env var, stdio is trusted transport
4. **No governance mutation** — governance state is read-only, `governance_ref` is write-once
5. **No execution path through Conductor** — Conductor coordinates, does not execute
6. **No competing authority state** — Solvent is authority, Conductor observes
7. **No domain-specific leakage** — Conductor is domain-agnostic, Solvent-specific logic stays in adapter
8. **Domain-agnostic Conductor model unchanged** — no project membership, no RBAC, no multi-tenancy

## Test Commands

```bash
rtk go build ./...
rtk go test ./...
```

## Expected Test Count

Current: 64 tests passing.
After changes: ~72-75 tests (5 new release tests + governance consistency tests + web binding test + MCP identity test).

## Output

After implementation:

- Changes made (per file)
- Tests run / results
- Remaining findings
- Explicit architecture conformance statement
- GREEN/YELLOW/RED verdict
