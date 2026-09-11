# Conductor Remediation Plan (v2 — Corrected)

Focused remediation of implementation defects identified by adversarial review.
No architecture redesign. No Solvent changes. No authorization engine.
No domain concepts. Locked architecture preserved.

```
AI / Coding / Research Agent = AGENCY
Conductor                  = COORDINATION
Solvent                    = AUTHORITY
External Executor          = EFFECT

CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION
```

---

## V1 Resource Access Model (Explicit Scope)

Conductor v1 is a **single trusted coordination instance/workspace**.

```
Authentication
    establishes caller identity (WHO is calling Conductor)
    does NOT establish per-project authorization

Resource IDs (UUIDs)
    reduce enumeration/discovery risk
    are NOT authorization boundaries

Conductor access control ≠ Solvent authority
```

There is no multi-tenancy, no project membership, no RBAC.
An authenticated caller has access to the Conductor instance's
projects and tasks. Cross-project isolation is explicitly out of scope.

If project-level isolation is needed later, that is a future capability,
not something smuggled into this remediation.

---

## Lifecycle State Machine (Corrected)

The official Conductor task lifecycle, including the release transition:

```
proposed  ──claim──>  active
active    ──submit──> review
active    ──block──>  blocked
active    ──cancel──> cancelled
active    ──release──> proposed      <-- NEW: recovery transition

review    ──accept──> accepted (terminal)
review    ──reject──> active
review    ──cancel──> cancelled

blocked   ──resolve──> active
blocked   ──cancel──> cancelled

accepted  (terminal)
cancelled (terminal)
```

Release transition:
- trigger: assigned agent releases task (abandonment recovery)
- actor: the agent currently assigned to the task
- from: active
- to: proposed
- effect: current_agent = NULL
- activity: task.released

No other lifecycle expansion.

---

## P0 — MUST FIX

### 1. Real HTTP Authentication

**Problem:** `Auth` middleware exists in `internal/api/auth.go` but is NOT applied
to the HTTP router. Additionally, the current implementation accepts ANY string
as an API key — wiring the middleware alone turns "no auth" into "any string = auth."

**Correction applied:** Establish a real credential boundary. Do not merely wire
the existing any-string-accepting middleware.

**Changes:**

| File | Change |
|---|---|
| `internal/api/auth.go` | Replace current any-string-accepting middleware. Add `NewAuthMiddleware(credentials map[string]string)` where `credentials` maps API key to stable Conductor actor ID. Use `crypto/subtle.ConstantTimeCompare` for key validation. Unknown/missing key -> 401. Credentials authenticate identity; credentials are not themselves identity |
| `internal/api/auth.go` | Remove `deriveAgentID` (SHA-256 hash as identity). Identity comes from the configured map, not from the credential |
| `internal/api/server.go` | Add `Credentials map[string]string` to `Server` struct. Wrap `s.dispatch` with `NewAuthMiddleware(s.Credentials)` in `ListenAndServe` |
| `cmd/conductor/main.go` | Read `CONDUCTOR_API_KEY` env var. Format: `key1:agent-1,key2:agent-2` (colon maps key to identity, comma separates entries). If empty: log fatal (POC requires at least one credential). Pass configured map to `api.NewServer` |
| `internal/api/task_handler.go` `CreateTask` | Do NOT strip status/current_agent. Validate and reject if present (see Finding 2) |
| `internal/api/api_test.go` | Fix `TestAPIAuthentication`: assert 401 on missing key. Add test for unknown key -> 401. Add test for known key -> authenticated |

**Authentication model:**
```
configured credentials:
    map[api_key]actor_id
    e.g., "secretA" -> "agent-1"
          "secretB" -> "agent-2"

request arrives
    |
extract key from X-API-Key or Authorization: Bearer
    |
key empty? -> 401 unauthorized
    |
key in configured map? -> NO -> 401 unauthorized
    |
YES -> look up actor_id from map
    |
store actor_id in request context
    |
handler uses configured identity, never caller-supplied identity
```

Principle: **Credentials authenticate identity; credentials are not themselves identity.**

**Tests:**
- `TestAPIUnauthenticatedRequestRejected` -- no API key -> 401
- `TestAPIInvalidCredentialRejected` -- unknown API key -> 401
- `TestAPIEmptyKeyRejected` -- empty string key -> 401
- `TestAPIAuthenticatedRequestDerivesIdentity` -- known key -> verify `current_agent` matches configured actor ID
- `TestAPICallerSuppliedIdentityIgnored` -- configured identity always used, never body-supplied

---

### 2. Task Creation Lifecycle Bypass — REJECT, Do Not Strip

**Problem:** `TaskRepo.Create` defaults status to "proposed" if empty, but accepts
any status the caller provides. HTTP `CreateTask` decodes full `domain.Task` from
body including `Status` and `CurrentAgent`.

**Correction applied:** Reject invalid input at both handler and repository levels.
Do NOT silently strip values.

**Behavior:**
```
request contains status or current_agent fields
    → handler rejects with lifecycle-bypass error (400)

repository Create() independently enforces:
    status must be empty or "proposed"
    current_agent must be nil
    → returns ErrLifecycleBypass if violated
```

**Changes:**

| File | Change |
|---|---|
| `internal/api/task_handler.go` `CreateTask` | After decoding, check: if `task.Status != "" && task.Status != "proposed"` → 400 lifecycle-bypass error. If `task.CurrentAgent != nil` → 400 lifecycle-bypass error. Do NOT strip — reject |
| `internal/store/task_repo.go` `Create` | Enforce: if `task.Status != "" && task.Status != TaskStatusProposed` → return `ErrLifecycleBypass`. If `task.CurrentAgent != nil` → return `ErrLifecycleBypass`. Then set defaults: `Status = TaskStatusProposed`, `CurrentAgent = nil` |
| `internal/mcp/tools.go` `createTask` | Already constructs task with only safe fields — no change. Add test |
| `internal/api/api_test.go` | Add rejection tests |
| `internal/store/store_test.go` | Add repo-level rejection tests |

**Tests:**
- `TestHTTPCreateTaskRejectsNonProposedStatus` — POST with `"status": "active"` → 400
- `TestHTTPCreateTaskRejectsCurrentAgent` — POST with `"current_agent": "x"` → 400
- `TestRepoCreateRejectsNonProposedStatus` — direct repo call with bad status → ErrLifecycleBypass
- `TestRepoCreateRejectsCurrentAgent` — direct repo call with current_agent → ErrLifecycleBypass
- `TestRepoCreateAcceptsProposedStatus` — explicit `"status": "proposed"` → accepted
- `TestRepoCreateAcceptsEmptyStatus` — empty status → defaults to proposed
- `TestMCPCreateTaskAlwaysProposed` — MCP create → proposed
- `TestClaimOnlyPathToAssignment` — only `Claim()` sets current_agent

---

### 3. Fix Solvent GetState

**Problem:** `Adapter.GetState` returns unconditional `"unknown"` with
`"adapter requires reference-specific routing"`. The working `GetBeliefState`
method is never called from `GetState`.

**Decision:** Missing `belief_id` → explicit error. `"unknown"` reserved for
genuinely indeterminate/unavailable state, not malformed provider references.

**Changes:**

| File | Change |
|---|---|
| `internal/adapter/solvent/adapter.go` `GetState` | Extract `belief_id` from `ref.Metadata["belief_id"]`. If present: call `client.GetBeliefExplain(ctx, ref.ReferenceID, beliefID)` then `translateBeliefExplain`. If missing: return explicit error (`"solvent adapter: belief_id required in governance reference metadata"`). Remove placeholder return |

**Tests:**
- `TestAdapterGetStateWithBeliefRef` — mock Solvent server, verify translated state
- `TestAdapterGetStateMissingBeliefID` — metadata lacks `belief_id` → explicit error
- `TestAdapterGetStateMalformedResponse` — mock returns invalid JSON → safe failure
- `TestAdapterGetStateSolventUnavailable` — mock server down → error propagation
- `TestAdapterGetStateNoAggregateInference` — verify no counter-based state inference

---

### 4. Wire GovernanceService / Provider Routing

**Problem:** `GovernanceService` implemented in `service/governance_service.go`
but `main.go:33` hardcodes `&governance.NullReader{}`. Provider routing is dead
infrastructure.

**Changes:**

| File | Change |
|---|---|
| `cmd/conductor/main.go` | Create `GovernanceService`. Register `"null"` to `NullReader`. If Solvent env vars (`SOLVENT_URL`, `SOLVENT_API_KEY`) are set: create `solvent.Client` + `solvent.Adapter`, register `"solvent"`. Pass `GovernanceService` to `api.NewServer` and `mcp.NewServer` instead of raw `NullReader` |

**Tests:**
- `TestGovernanceServiceRoutesToSolvent` — register solvent provider, verify `GetState` reaches adapter
- `TestGovernanceServiceNullDefault` — null provider works through service
- `TestGovernanceServiceUnknownProvider` — already exists, verify still passes

---

### 5. Protect governance_ref from Arbitrary Mutation

**Problem:** `TaskUpdateFields` in `task_repo.go:129-135` includes
`GovernanceRef *string`. HTTP PATCH and MCP `updateTask` allow setting it after
creation.

**Changes:**

| File | Change |
|---|---|
| `internal/store/task_repo.go` `TaskUpdateFields` | Remove `GovernanceRef` field entirely |
| `internal/store/task_repo.go` `Update` | Remove the `GovernanceRef` code block (lines 97-100) |
| `internal/store/task_repo.go` `Create` | Still accepts `GovernanceRef` — write-once at creation |

**Tests:**
- `TestTaskCreateWithGovernanceRef` — create with governance_ref → stored correctly
- `TestTaskUpdateCannotMutateGovernanceRef` — PATCH attempt → unchanged
- `TestMCPUpdateCannotMutateGovernanceRef` — MCP update with governance_ref → ignored
- `TestRepositoryUpdateIgnoresGovernanceRef` — direct `Update()` with GovernanceRef → field ignored

---

## P1 — MUST FIX BEFORE REAL POC USE

### 6. Fix dependency-aware next_task

**Problem:** `nextTask` in `tools.go:296-314` returns first task with
`Status == proposed` regardless of dependencies.

**Changes:**

| File | Change |
|---|---|
| `internal/mcp/server.go` | Add `DependencyRepo *store.DependencyRepository` field to `Server` struct. Initialize in `NewServer` |
| `internal/mcp/tools.go` `nextTask` | For each candidate with `Status == proposed && CurrentAgent == nil`: list dependencies via `DependencyRepo.ListByTask`, check each `blocked_by_id` task status. If any blocker is not `accepted` or `cancelled`, skip. Return first fully-resolved task |

**Dependency resolution semantics (existing model):**
- `blocked_by_id` task status `accepted` → resolved
- `blocked_by_id` task status `cancelled` → resolved (cancelled dependency permits dependent)
- `blocked_by_id` task status `proposed/active/review/blocked` → unresolved

**Tests:**
- `TestNextTaskSkipsTaskWithUnresolvedDependency` — task A blocked by B (proposed) → A skipped
- `TestNextTaskReturnsEligibleTask` — task A blocked by B (accepted) → A returned
- `TestNextTaskDeterministicOrder` — multiple eligible → deterministic return (by created_at)
- `TestNextTaskCancelledDependencyPermits` — blocker cancelled → dependent eligible
- `TestNextTaskNoDependencies` — no deps → eligible

---

### 7. Task/Project Access Boundaries (Explicit Scope)

**Decision:** Conductor v1 is single-workspace/trusted-instance.

**Do NOT introduce multi-tenancy, project membership, or RBAC.**

```
Conductor v1 resource access model:

Authentication
    establishes caller identity

Resource IDs (UUIDs)
    reduce enumeration risk
    are NOT authorization boundaries

Single workspace
    authenticated callers access the instance's projects/tasks
    no per-project authorization in v1

Cross-project access control
    explicitly OUT OF SCOPE for v1
```

**Changes:** No code changes for this finding. The existing model is correct
for v1 scope. Document this explicitly.

**What we do NOT add:**
- Project membership tables
- Per-project access checks
- RBAC or capability matching
- Authorization engine

**What provides resource integrity:**
- Authenticated boundary (Finding 1)
- Lifecycle enforcement (Finding 2)
- Immutable governance_ref (Finding 5)
- Dependency same-project validation (existing)
- Unpredictable UUIDs (Finding 9) — reduces enumeration, not authorization

---

### 8. Abandoned-Task Recovery (Formalized Lifecycle Transition)

**Problem:** `Release()` exists in `task_repo.go:279-318` but has no HTTP endpoint
or MCP tool. The lifecycle does not officially contain `active → proposed`.

**Decision:** Release only. No `reassign_to`. Reassignment = release + claim.

**Lifecycle change (minimal):**
```
Add to allowed transitions:
    active → proposed (release)

validateTransition() must include this transition.
domain/task.go CanTransitionTo must include this transition.
```

**Changes:**

| File | Change |
|---|---|
| `internal/domain/task.go` `CanTransitionTo` | Add `TaskStatusActive: {TaskStatusReview, TaskStatusBlocked, TaskStatusCancelled, TaskStatusProposed}` |
| `internal/store/task_repo.go` `validateTransition` | Add `domain.TaskStatusActive: {domain.TaskStatusReview, domain.TaskStatusBlocked, domain.TaskStatusCancelled, domain.TaskStatusProposed}` |
| `internal/store/task_repo.go` `Release` | Already implements the SQL-level logic. Verify it aligns with the formalized transition |
| `internal/api/routes.go` `dispatch` | Add `POST /v1/tasks/{id}/release` case |
| `internal/api/task_handler.go` | Add `ReleaseTask` handler: extract task ID, get agent from context, call `TaskRepo.Release(ctx, taskID, agentID)`, return updated task |
| `internal/mcp/tools.go` | Add `conductor_release_task` tool handler |
| `internal/mcp/server.go` `handleListTools` | Register `conductor_release_task` in tools list |
| `internal/store/store_test.go` | Update lifecycle tests to include active → proposed |
| `internal/store/store_test.go` | Add invalid transition test: proposed → proposed should still be invalid |

**Tests:**
- `TestReleaseByAssignedAgent` — assigned agent releases → task proposed, current_agent nil
- `TestReleaseByOtherAgentRejected` — different agent → error
- `TestReleaseActivityRecorded` — activity "task.released" appended
- `TestReleaseLifecycleValid` — active → proposed via Release is valid
- `TestInvalidTransitionProposedToProposed` — still invalid (no self-transition)
- `TestMCPReleaseTask` — MCP tool works correctly

---

### 9. Replace Fake UUID Generation

**Problem:** `generateUUID()` at `task_repo.go:379-381` uses
`time.Now().UnixNano()` with modulo — deterministic and collision-prone.
MCP uses `task-{projectID}` fallback.

**Changes:**

| File | Change |
|---|---|
| `go.mod` | Move `github.com/google/uuid` from indirect to direct dependency |
| `internal/store/task_repo.go` `generateUUID` | Replace body with `uuid.New().String()` |
| `internal/mcp/tools.go` `createTask` | Replace `fmt.Sprintf("task-%s", projectID)` with `uuid.New().String()` |
| `internal/mcp/tools.go` `postActivity` | Replace `fmt.Sprintf("act-%d", time.Now().UnixNano())` with `uuid.New().String()` |

**Tests:**
- `TestUUIDUniqueness` — generate 10000 IDs → all unique
- `TestUUIDFormat` — valid UUID v4 format (8-4-4-4-12 hex)
- `TestConcurrentUUIDGeneration` — concurrent generation → no collisions

---

## P2 — REVIEW / TIGHTEN

### 10. Activity Action Semantics

**Problem:** Activities are append-only (correct). `"task.released"` action is
used in `Release()` but has no constant in `domain/activity.go`. Activity must
remain non-authoritative.

**Changes:**

| File | Change |
|---|---|
| `internal/domain/activity.go` | Add `ActionTaskReleased = "task.released"` constant |

No other changes. Activity remains generic, append-only, observational.
No domain ontology. No restriction on action strings.

### 11. CheckAuthorization — Reserved Governance Capability

**Problem:** `CheckAuthorization` is part of the `GovernanceReader` interface but
has no production consumer. It risks becoming dead infrastructure or being confused
with Conductor authorization.

**Decision:** Keep `CheckAuthorization` as a reserved read-only governance
capability. It is part of the `GovernanceReader` contract. The distinction
is useful:

```
GetState()
    -> what is the current governance state?

CheckAuthorization()
    -> what does the external authority currently say
      about this proposed consequence?
```

Neither creates Conductor authority. Both are informational queries.

`CheckAuthorization` is **reserved/internal** — not surfaced through HTTP/MCP
in v1. This avoids confusion with Conductor authorization while preserving the
governance contract for future projections that need it.

**Changes:**

| File | Change |
|---|---|
| `internal/governance/reader.go` | Add doc comment: `CheckAuthorization is a reserved read-only governance capability. Its result is informational, never persisted as Conductor truth, never used to authorize execution, and never constitutes Conductor authorization` |
| `internal/governance/types.go` `AuthorizationResult` | Add doc comment reinforcing informational-only semantics |
| `internal/governance_handler.go` | No change -- `GetGovernance` handler does not call `CheckAuthorization` (it only calls `GetState`). This is correct for v1 |
| `internal/adapter/solvent/adapter.go` `CheckAuthorization` | Already implemented and tested. No change |
| `internal/service/governance_service.go` `CheckAuthorization` | Already implemented and tested. No change |

**Tests:** Existing tests in `governance_service_test.go` and
`adapter_test.go` already cover `CheckAuthorization`. No new tests needed.

---

## Files Changed Summary

| File | Findings |
|---|---|
| `cmd/conductor/main.go` | 4 |
| `internal/api/server.go` | 1 |
| `internal/api/auth.go` | 1 |
| `internal/api/routes.go` | 8 |
| `internal/api/task_handler.go` | 1, 2, 8 |
| `internal/api/api_test.go` | 1, 2 |
| `internal/domain/task.go` | 8 |
| `internal/domain/activity.go` | 10 |
| `internal/store/task_repo.go` | 2, 5, 8, 9 |
| `internal/store/store_test.go` | 2, 5, 8, 9, 10 |
| `internal/adapter/solvent/adapter.go` | 3 |
| `internal/adapter/solvent/adapter_test.go` | 3 |
| `internal/governance/reader.go` | 11 |
| `internal/governance/types.go` | 11 |
| `internal/mcp/server.go` | 6, 8 |
| `internal/mcp/tools.go` | 6, 8, 9 |
| `internal/mcp/mcp_test.go` | 6, 8 |
| `go.mod` | 9 |

---

## Verification Checklist

After implementation, verify:

1. HTTP auth uses real credential boundary — unknown key → 401, known key → authenticated identity
2. MCP trusted-local behavior consistent — same invariants as HTTP
3. Task creation REJECTS lifecycle/assignment fields — does not strip them
4. Generic task update cannot mutate lifecycle fields or governance_ref
5. governance_ref immutable after creation
6. Real Solvent reference-specific state reaches GetGovernance path
7. GovernanceService provider routing is wired and functional
8. No aggregate Solvent statistics used for inference
9. next_task respects dependencies
10. Task IDs are collision-resistant UUIDs
11. Abandoned tasks recoverable via release endpoint/tool
12. `active → proposed` (release) is in the formal lifecycle
13. Activity remains append-only and non-authoritative
14. CheckAuthorization is a reserved governance capability, not Conductor authorization
15. Conductor contains no execution path
16. Conductor contains no competing authority state
17. No domain-specific concepts entered the core
18. Conductor v1 is explicitly single-workspace — no per-project authorization

---

## Architectural Boundary Check

After remediation, verify this model holds:

```
AGENT
  reason / propose / act / report
        |
        v
CONDUCTOR
  project / task / dependency / assignment / lifecycle / activity
        |
        | read-only governance observation
        v
GOVERNANCEREADER
        |
        v
SOLVENT
  authority / target / state binding / authorization / revocation / claim
        |
        v
EXTERNAL EXECUTOR
  actual external effect
```

- Agent may interact directly with Solvent/external systems for consequential actions
- Conductor must never become part of that execution path
- Conductor access control ≠ Solvent authority
- Conductor authentication = identity only, not authorization
- Conductor lifecycle = coordination rules, not execution authority

---

## Corrections Applied (v1 → v2)

| # | Correction | Applied To |
|---|---|---|
| 1 | Authentication must be real — configured credential set, not any-string | Finding 1: full rewrite of auth approach |
| 2 | Task creation must REJECT, not silently strip | Finding 2: handler rejects + repo enforces |
| 3 | Explicitly define v1 single-workspace model | New section + Finding 7 rewritten |
| 4 | Formalize active → proposed release transition | Finding 8: lifecycle update + validateTransition + tests |
| 5 | CheckAuthorization — reserved governance capability | New Finding 11 |
| 6 | Auth identity: credential map, not SHA-256 hash as identity | Finding 1: identity from config, not credential |
| 7 | Preserve all architectural boundaries | Throughout, reinforced in verification |

---

## Final Verdict Target

GREEN — REMEDIATION COMPLETE

Requires:
- No CRITICAL findings
- No HIGH findings
- No lifecycle bypass
- No identity/authentication bypass
- No competing Conductor/Solvent authority
- No execution path through Conductor
- Functional reference-specific Solvent observation
- Domain-agnostic core preserved
- Real credential boundary
- Formalized release transition
- Documented single-workspace scope
