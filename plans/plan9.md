# plan9.md — Targeted Remediation: Governance Ref Creation Clarity + Observation Timestamp

Status: **PLANNED — not yet implemented**
Source: `adv_review3.md` (third adversarial review)
Review verdict: GREEN architecture, two interaction-level ambiguities

---

## Findings

| # | Finding | Status |
|---|---------|--------|
| 1 | HTTP `CreateTask` accepts `governance_ref` implicitly via JSON decode — works but undocumented | Needs explicit guard + comment |
| 2 | MCP `create_task` tool does NOT accept `governance_ref` — schema and implementation both omit it | **Real gap** |
| 3 | Store write-once semantics already enforced — `TaskUpdateFields` excludes it, lifecycle transitions never touch it | Already correct |
| 4 | `GovernanceState.RefreshedAt` field already exists in the type definition | Already correct |
| 5 | Web UI governance panel does NOT render `RefreshedAt` | **Real gap** |
| 6 | Governance error fallback sets zero-value `RefreshedAt` (no timestamp) | Minor — set `time.Now()` on fallback |
| 7 | No test exercises `governance_ref` through HTTP or MCP creation | **Real gap** |
| 8 | No test verifies `governance_ref` immutability via PATCH or lifecycle | **Real gap** |

---

## Change 1: MCP `create_task` — accept `governance_ref`

**Files:**
- `internal/mcp/server.go` — add `governance_ref` property to schema (optional, not required)
- `internal/mcp/tools.go` — extract `governance_ref` from args, set on task before Create

The description should read: `"Opaque governance reference (optional, write-once at creation)"`.

**Schema change (server.go):**
```go
// In conductor_create_task inputSchema properties:
"governance_ref": map[string]interface{}{
    "type":        "string",
    "description": "Opaque governance reference (optional, write-once at creation)",
},
```

**Implementation change (tools.go):**
```go
// In createTask, after extracting description:
govRef, _ := args["governance_ref"].(string)
if govRef != "" {
    task.GovernanceRef = &govRef
}
```

---

## Change 2: HTTP API `CreateTask` — explicit documentation comment

**File:** `internal/api/task_handler.go`

No code change needed — `TaskUpdateFields` already prevents PATCH from modifying `governance_ref`. Add a one-line comment above the CreateTask decode block:

```go
// governance_ref may be supplied at creation time. Once set, it is immutable.
```

---

## Change 3: Governance observation timestamp in UI

**File:** `internal/web/templates/layout.html`

Add `RefreshedAt` rendering to the governance panel, after the status line:

```html
{{if not .GovernanceState.RefreshedAt.IsZero}}
<p>As of {{.GovernanceState.RefreshedAt.UTC.Format "2006-01-02 15:04 UTC"}}</p>
{{end}}
```

Renders "As of 2026-09-11 10:42 UTC" when timestamp is set, nothing when zero (error fallback).

---

## Change 4: Set `RefreshedAt` on governance error fallback

**Files:**
- `internal/api/governance_handler.go` — set `RefreshedAt: time.Now()` in error fallback
- `internal/mcp/tools.go` — same in `getGovernance` error fallback

Fallback becomes:
```go
state = &governance.GovernanceState{
    Reference:   ref,
    Status:      governance.GovernanceStatusUnknown,
    Blockers:    []string{"provider unavailable"},
    RefreshedAt: time.Now(),
}
```

---

## Change 5: Tests

**File:** `internal/store/store_test.go` — add:

1. `TestTaskCreateWithGovernanceRef` — create task with governance_ref, verify round-trip
2. `TestTaskGovernanceRefImmutabilityViaUpdate` — create with governance_ref, PATCH title, verify governance_ref unchanged
3. `TestTaskGovernanceRefImmutabilityViaLifecycle` — create with governance_ref, claim/submit/release, verify governance_ref unchanged throughout

**File:** `internal/api/api_test.go` — add:

4. `TestHTTPCreateTaskWithGovernanceRef` — POST with governance_ref in body, verify 201 and persisted value
5. `TestHTTPCreateTaskWithoutGovernanceRef` — POST without governance_ref, verify 201 and nil governance_ref
6. `TestHTTPUpdateTaskDoesNotAlterGovernanceRef` — create with governance_ref, PATCH title, verify governance_ref unchanged

**File:** `internal/mcp/mcp_test.go` — add:

7. `TestMCPCreateTaskWithGovernanceRef` — call create_task with governance_ref arg, verify response contains it
8. `TestMCPCreateTaskWithoutGovernanceRef` — call create_task without arg, verify nil

---

## Change 6: Documentation micro-edits

- `internal/api/task_handler.go` — one-line comment on CreateTask
- `internal/mcp/tools.go` — one-line comment on createTask

No AGENTS.md rewrite. The existing §11 already states the creation-time rule. The comment is for implementer visibility, not architecture.

---

## What does NOT change

- Domain model (`internal/domain/task.go`)
- `TaskUpdateFields` (`internal/store/task_repo.go`) — already correct
- Lifecycle transitions — already correct
- `GovernanceReader` interface — no changes
- `CheckAuthorization` — stays absent
- `GovernanceState` type — already has `RefreshedAt`
- AGENTS.md — no rewrite (§11 already covers the rule)
- No new core primitives, no new authority paths, no new domain concepts

---

## Open Question

The MCP `conductor_create_task` schema currently also omits `task_id` (which the implementation accepts as optional at `tools.go:94`). Should I add `task_id` to the schema too while I'm in there, or keep the scope strictly to `governance_ref` only?

**Awaiting decision before implementation.**

---

## Validation

After all changes:

1. `go build ./...`
2. `go test ./...` — existing 69 + new tests
3. `go vet ./...`

---

## Regression / Architecture Check

After implementation, diff inspection must verify:

- No new authorization path exists
- `GovernanceReader` remains read-only
- No `CheckAuthorization` method reintroduced
- Conductor never authorizes consequences
- Conductor never executes external effects
- Governance remains optional to ordinary coordination
- Solvent remains external governance/authority
- `governance_ref` remains opaque to Conductor
- `governance_ref` remains immutable after task creation
- No domain-specific concepts entered Conductor core
- No new Conductor core primitive introduced
- No new background polling or live-authorization mechanism introduced
- No unrelated architecture changes made
