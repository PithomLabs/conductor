# CONDUCTOR — COMPREHENSIVE ADVERSARIAL REVIEW

**Repository:** `/home/chaschel/Documents/go/conductor`
**Review Date:** 2026-09-11
**Reviewer Role:** Senior Go Architect / Adversarial Reviewer
**Mode:** Architectural boundary enforcement. No modifications performed.

---

## EXECUTIVE SUMMARY

The Conductor implementation is structurally sound in its data model and domain-agnosticity. The lifecycle state machine, cross-project isolation, and execution boundary are correctly implemented. However, the review uncovered **critical failures in identity, authentication, and authorization enforcement** that allow a malicious agent to exploit every major surface. The `Auth` middleware exists but is never wired into the HTTP router. The MCP stdio server has zero authentication. Task creation accepts `current_agent` directly, allowing claim bypass. The Solvent adapter's primary `GetState` is a non-functional stub. `GovernanceService` is instantiated nowhere. `CheckAuthorization` is dead code in the public interface. These are not test failures—they are architectural boundary collapses.

---

## 1. ARCHITECTURAL BOUNDARY AUDIT

### Findings by Package

| Package | Declared Responsibility | Actual Behavior | Verdict |
|---|---|---|---|
| `internal/domain/` | Pure data model | Clean. Generic only. | PASS |
| `internal/store/` | Persistence | Clean. Lifecycle enforcement in repo layer. | PASS (with transaction gaps) |
| `internal/api/` | Agent-facing HTTP API | Has `Auth` middleware, but **never applied to any route**. All endpoints unauthenticated. | FAIL |
| `internal/mcp/` | Agent-facing MCP | Zero authentication. Fixed `agentID` at construction. | FAIL |
| `internal/web/` | Minimal human UI | Zero authentication. All project/task data exposed. | FAIL |
| `internal/service/` | Governance projection service | **Dead code.** Instantiated nowhere. `GovernanceService` not wired into API or web. | FAIL |
| `internal/governance/` | Read-only governance abstraction | Interface and types are clean. `CheckAuthorization` is informational. | PASS (with dead-code risk) |
| `internal/adapter/solvent/` | Solvent integration | `GetState` is a **non-functional stub** always returning `unknown`. Real query is `GetBeliefState`, unexported and unreachable through the `GovernanceReader` interface. | FAIL |
| `internal/agentstub/` | Test/demo agents | Stubs are test-only. `Release` method in repo is **unreachable** from any surface. | PASS (test scope) |

### Hidden Authority Duplication

- **`CheckAuthorization` in the public `GovernanceReader` interface** — This method is part of the exported interface, implemented by the adapter, and exposed by `GovernanceService`, but is **never called by any production code path**. It creates a false impression that Conductor performs authorization checks. A future developer could easily start calling it and treating its result as permission.

- **`governance_ref` on `conductor_task`** — The field is correctly opaque JSON. However, the `Update` API handler allows arbitrary mutation of `governance_ref`. An adversarial agent can redirect the governance reference to a fake provider, then query it via `GetGovernance`. Conductor would faithfully project whatever the fake provider returns, creating a **shadow authority state** under attacker control.

### Verdict

The architectural boundary between CONDUCTOR and SOLVENT is breached at the API surface. The `Auth` middleware is defined but dead. The MCP surface is unauthenticated. The Solvent adapter's primary query method is a stub. `GovernanceService` is unused infrastructure. These are not design issues—they are implementation gaps that a malicious agent will exploit.

---

## 2. DOMAIN-AGNOSTICITY AUDIT

The core model (Project, Task, Activity, Dependency) is genuinely generic. No domain ontology is encoded in status values, lifecycle transitions, or activity actions. The `governance_ref` is correctly opaque JSON. The `agentstub` tests confirm domain extension readiness for restaurant reservations and library systems.

**One subtle domain leak:** The `next_task` MCP tool and its equivalent logic in the stub agents return the first proposed task in creation order. For domain workloads where dependency ordering matters (e.g., scientific computing pipelines), this is a **coordination failure**, not a domain encoding. It is a correctness bug in the coordination logic, not a domain assumption.

**Verdict:** PASS. Domain-agnosticity is preserved. The `next_task` ordering issue is a coordination bug, not a domain leak.

---

## 3. LIFECYCLE STATE-MACHINE AUDIT

### Valid Transitions (from `task_repo.go`)

```
proposed  -> active   (Claim)
proposed  -> blocked  (ReportBlocker)
proposed  -> cancelled
active    -> review   (Submit)
active    -> blocked  (ReportBlocker)
active    -> cancelled
review    -> accepted (Accept)
review    -> active   (Reject)
review    -> cancelled
blocked   -> active   (ResolveBlocker)
blocked   -> cancelled
```

### Critical Bypass: `current_agent` set at creation

`TaskRepository.Create` accepts the full `Task` struct including `CurrentAgent`. The HTTP API `CreateTask` handler decodes the request body directly into a `domain.Task` and passes it to `Create` without stripping `CurrentAgent`. An adversarial client can:

```json
POST /v1/projects/proj-1/tasks
{
    "id": "evil-task",
    "title": "Bypass Claim",
    "current_agent": "attacker"
}
```

This creates an `active` task (default status is `proposed` but `current_agent` is set). The attacker then controls the task without ever invoking `Claim`. The `Update` endpoint correctly excludes `status` and `current_agent`, but the **creation path is unprotected**.

**Impact:** An adversarial agent can pre-assign tasks to itself, bypassing the coordination mechanism entirely.

### Critical Bypass: `status` set at creation

While the default status is `proposed`, the `Create` method does not enforce that `current_agent` is nil when `status` is `proposed`, nor does it enforce that `current_agent` is non-nil when `status` is `active`. A task can be created with `status: "active"` and `current_agent: nil`, or `status: "proposed"` and `current_agent: "attacker"`.

### Transition Validation Strength

The `validateTransition` function and the `Transition` method correctly enforce state machine rules using optimistic checks (`currentStatus != fromStatus`). This is **not atomic under concurrent modification** in the sense that two concurrent callers could both pass the optimistic check before either commits, but SQLite's serializable isolation for WAL mode plus the single-writer constraint makes this practically safe. The `Claim` method uses a conditional UPDATE (`WHERE status = 'proposed' AND current_agent IS NULL`) which is properly atomic.

### Terminal State Enforcement

Terminal states (`accepted`, `cancelled`) correctly reject all outgoing transitions. The `Transition` method returns `ErrInvalidTransition` for terminal-to-any transitions.

### Verdict

The lifecycle state machine has a **critical bypass in task creation** that allows `current_agent` and `status` to be set directly, circumventing the `Claim` mechanism. Transition validation is correct for post-creation mutations.

---

## 4. TRANSACTIONAL INTEGRITY

### Atomicity of State + Activity

The `Transition`, `TransitionWithAgent`, and `Claim` methods correctly wrap state update + activity insert in a single database transaction via `db.Transaction()`. If either step fails, both roll back. This is correct.

### Critical Gap: `Update` is NOT transactional

`TaskRepository.Update` executes multiple SQL statements (one per field) **without a transaction wrapper**. If the process crashes between setting `title` and setting `updated_at`, the database is left with a partially updated row. This is not a correctness bug for the current single-row update pattern (each statement auto-commits), but it becomes a real failure mode if `Update` is extended to update multiple rows or if a future change adds related writes.

### Critical Gap: `Create` is NOT transactional

`TaskRepository.Create` and `ProjectRepository.Create` execute a single INSERT, which is atomic at the SQLite level. This is acceptable for single-row inserts, but if future changes add related writes (e.g., creating default activities), the lack of a transaction wrapper becomes a consistency risk.

### UUID Collision Risk

`generateUUID()` in `task_repo.go`:
```go
func generateUUID() string {
    return fmt.Sprintf("%d-%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%10000, time.Now().UnixNano()%100000)
}
```

This is **not a UUID**. It is timestamp-based with a small random suffix. Under high concurrency (multiple goroutines creating activities in the same nanosecond), collisions are possible. Colliding activity IDs would cause INSERT failures, breaking the state+activity atomicity guarantee.

### Concurrency on Claim

`Claim` uses a conditional UPDATE:
```sql
UPDATE conductor_task SET current_agent = ?, status = 'active', updated_at = datetime('now')
WHERE id = ? AND status = 'proposed' AND current_agent IS NULL
```

This is properly atomic. Two concurrent claimers: one wins, one gets `RowsAffected == 0`. Correct.

### Verdict

Transaction wrapping is correct for lifecycle transitions but absent for generic updates. UUID generation is a real collision risk under concurrency. These are implementation gaps, not architectural violations.

---

## 5. IDENTITY / AUTHENTICATION AUDIT

### Critical Finding: `Auth` Middleware is Dead Code

The `Auth` middleware exists in `internal/api/auth.go`:

```go
func Auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        // ... derives agentID from apiKey ...
    })
}
```

However, it is **never applied to any route** in `internal/api/routes.go`:

```go
func (s *Server) registerRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/v1/", s.dispatch)  // No middleware chain
}
```

**Impact:** Every HTTP endpoint is completely unauthenticated. The `agentIDFromContext` function returns `"unknown"` for all requests. An adversarial client can invoke `ClaimTask`, `SubmitTask`, `AcceptTask`, `RejectTask`, `ReportBlocker`, `ResolveBlocker`, `PostActivity`, and `UpdateTask` without any identity.

### Critical Finding: MCP Has Zero Authentication

The MCP stdio server (`internal/mcp/server.go`) reads JSON from stdin. The `agentID` is fixed at construction:

```go
case "mcp":
    agentID := "agent-mcp"  // hardcoded in main.go
    server := mcp.NewServer(db, gov, agentID)
```

Any process with write access to the MCP server's stdin can invoke any tool as `agent-mcp`. There is no challenge-response, no API key, no identity verification.

### Weak API Key Design

Even if `Auth` were wired in, the identity model is weak:
- API keys are not stored or verified server-side. Any string is accepted.
- `deriveAgentID` hashes the key with SHA-256 and takes 8 bytes. This is deterministic: the same key always maps to the same agent ID. But there is no registry of valid keys, so any random string produces a valid agent identity.
- There is no way to revoke or rotate keys.

### Actor Spoofing in Activity

In `PostActivity` (HTTP API):
```go
activity.ActorType = "agent"
activity.ActorID = agentID  // from context, which is unauthenticated
```

Since `Auth` is not applied, `agentID` is always `"unknown"`. In MCP:
```go
activity.ActorID = s.agentID  // fixed at construction
```

Both paths allow the caller to control the actor attribution. An adversarial agent can record activities claiming to be another agent or a human.

### Verdict

Authentication is entirely absent. The `Auth` middleware is defined but never used. The MCP surface is a wide-open pipe. Identity is either unauthenticated (`unknown`) or fixed at construction. This is a complete collapse of the identity boundary.

---

## 6. AUTHORIZATION AUDIT

### Critical Finding: No Authorization Exists

Conductor has **no authorization layer**. The `CheckAuthorization` method on `GovernanceReader` is the only authorization-adjacent code, and it is:
1. Dead code (never called by any production path)
2. Informational only (returns `AuthorizationResult` but the result is never used to gate any operation)

There is no check that:
- The calling agent is allowed to claim a specific task
- The calling agent is allowed to submit a task it claimed
- The calling agent is allowed to accept/reject a task (reviewer role)
- The calling agent is allowed to create tasks in a project
- The calling agent is allowed to update a task

### Task Assignment != Authorization

The `current_agent` field on Task is treated as assignment, not authorization. But since any unauthenticated caller can set `current_agent` at creation time, the assignment is meaningless.

### Governance Status != Authorization

The `GetGovernance` endpoint returns governance state, but no operation checks it. A task with `governance_ref` pointing to a `ready` state can be submitted, accepted, or rejected without any authorization check.

### Verdict

Conductor is not an authorization engine, and it correctly does not claim to be one. However, the absence of any authorization layer means that **any caller can perform any operation on any resource**. The `CheckAuthorization` dead code creates a dangerous false impression that authorization is being performed.

---

## 7. SOLVENT INTEGRATION AUDIT

### Critical Finding: `GetState` is a Non-Functional Stub

```go
func (a *Adapter) GetState(ctx context.Context, ref governance.GovernanceReference) (*governance.GovernanceState, error) {
    if ref.ReferenceID == "" {
        return nil, fmt.Errorf("reference_id is required")
    }
    // For now, we treat reference_id as scenario_id and query the first belief
    // In production, the reference would need to include belief_id or target_id
    // This is a limitation of the current Solvent API integration
    return &governance.GovernanceState{
        Reference: ref,
        Status:    governance.GovernanceStatusUnknown,
        Blockers:  []string{"adapter requires reference-specific routing"},
    }, nil
}
```

This **always returns `unknown`** with a blocker message. It never queries Solvent. The actual Solvent query logic lives in `GetBeliefState`, which is unexported and not part of the `GovernanceReader` interface.

**Impact:** The `GetGovernance` HTTP endpoint and the `conductor_get_governance` MCP tool always return `unknown` status, regardless of actual Solvent state. This is a complete failure of the governance projection.

### Critical Finding: `GovernanceService` is Dead Code

`internal/service/governance_service.go` implements provider routing:
```go
func (s *GovernanceService) GetState(ctx context.Context, ref governance.GovernanceReference) (*governance.GovernanceState, error) {
    reader, ok := s.readers[ref.Provider]
    // ...
}
```

But `GovernanceService` is **never instantiated** by `api.NewServer` or `web.NewServer`. Both use `NullReader` directly. The provider routing logic exists but is unreachable.

### Solvent-Specific Vocabulary Leak

The `BeliefExplainResponse` struct in `client.go` contains Solvent-specific fields:
```go
type BeliefExplainResponse struct {
    IsPromoted bool
    IsRetracted bool
    CanPromote bool
    PromotionBlockedReason string
    CanAuthorize bool
    AuthorizationBlockedReason string
    // ...
}
```

These are correctly confined to the `solvent` package. The `translateBeliefExplain` function maps them to generic governance states. This part is clean.

### Verdict

The Solvent integration is architecturally correct in structure but **functionally broken**. The primary `GetState` path is a stub. The provider-routing service is unused. A caller querying governance will always see `unknown`, which is indistinguishable from "Solvent is down" or "no governance configured."

---

## 8. EXECUTION BOUNDARY AUDIT

Conductor does not execute consequential actions. There are no shell execution helpers, no command runners, no external mutation paths beyond the Solvent adapter (which is read-only). The `agentstub` package's `performWork` sleeps. `RealAgent.performActualWork` prints messages. No execution occurs.

**One observation:** The `web` UI and `api` server both run HTTP listeners. While they don't execute consequential actions themselves, they expose the full coordination surface. An adversarial agent that compromises the API server could use it to manipulate task state. This is an operational security concern, not an architectural violation.

**Verdict:** PASS. The execution boundary is preserved. Conductor does not and cannot execute consequential actions.

---

## 9. ACTIVITY SEMANTICS AUDIT

Activities are append-only. The `ActivityRepository` has only `Append` (INSERT) and list operations. No UPDATE, DELETE, or mutation of existing records. The `conductor_activity` table has no UPDATE or DELETE paths in any code.

**Issue:** Activity actions are uncontrolled strings. The `post_activity` MCP tool and `PostActivity` HTTP endpoint accept any `action` string. An adversarial agent can record:
- `"task.executed"` — implying execution occurred
- `"task.verified"` — implying verification occurred
- `"task.approved"` — implying approval occurred
- `"deploy.completed"` — implying deployment occurred

While activities are documented as observational, a downstream system (or a human reading the UI) could be misled by these semantically loaded action strings.

**Issue:** The `details` field is documented as "JSON metadata (non-authoritative)" but is never validated. An adversarial agent can inject misleading details.

**Verdict:** PASS with caveats. Activity is append-only and non-authoritative. The uncontrolled action vocabulary is a semantic risk, not an architectural violation.

---

## 10. GOVERNANCE PROJECTION AUDIT

The governance projection is read-only. The `GetGovernance` HTTP endpoint and `conductor_get_governance` MCP tool both return read-only state. The UI labels it "Governance (read-only)" with a note: "Consequences are executed via external governance system, not through Conductor."

**Critical Issue:** Because `GetState` always returns `unknown`, the governance projection is **always stale or missing**. A caller cannot distinguish between:
1. No governance configured (NullReader returns `unknown` with "no external governance system configured")
2. Solvent is configured but `GetState` stub returns `unknown` with "adapter requires reference-specific routing"
3. Solvent is configured and reachable but the reference has no current state

All three cases produce the same `unknown` status. This makes the governance projection **operationally useless** and potentially **misleading** — a reviewer seeing `unknown` might assume no governance applies, when in fact Solvent has a definitive state that Conductor is failing to retrieve.

**Verdict:** The projection is correctly read-only and non-authoritative in design, but functionally broken in implementation. The UI wording is correct, but the underlying data is always `unknown`.

---

## 11. MCP ADVERSARIAL AUDIT

### Authentication: NONE

The MCP server has zero authentication. Any process that can write to stdin can invoke any tool. The `agentID` is fixed at construction (`"agent-mcp"` by default).

### Lifecycle Bypass via `conductor_create_task`

The MCP `create_task` tool accepts a `task_id` argument:
```go
func (s *Server) createTask(args map[string]interface{}) MCPResponse {
    taskID, _ := args["task_id"].(string)
    if taskID == "" {
        taskID = fmt.Sprintf("task-%s", projectID)  // predictable
    }
    // ...
}
```

The generated `taskID` is predictable (`task-{projectID}`). An adversarial agent can guess task IDs and manipulate them. More critically, the `createTask` MCP tool does not validate that the caller is allowed to create tasks in the project.

### Identity Spoofing via `conductor_post_activity`

The MCP `post_activity` tool records activity with `ActorID = s.agentID` (the fixed server agent ID). But the HTTP `PostActivity` endpoint uses `agentIDFromContext`, which returns `"unknown"` when `Auth` is not applied. Both surfaces allow the caller to influence actor attribution.

### Governance Mutation via `governance_ref` Update

The MCP `update_task` tool accepts `title` and `description` only — **not** `governance_ref`. This is correct. However, the HTTP `UpdateTask` endpoint accepts `governance_ref` via `TaskUpdateFields`. An adversarial client using HTTP can redirect governance references.

### Unauthorized Acceptance/Rejection

The MCP `conductor_accept_task` and `conductor_reject_task` tools have no role check. Any MCP client can accept or reject any task in review status. There is no verification that the caller is a "reviewer."

### Cross-Project Dependency Manipulation

The MCP `create_task` tool creates tasks in the specified project. The MCP `update_task` tool does not allow changing `project_id`. Cross-project isolation is preserved at the MCP level.

### Malformed Arguments

The MCP tools use Go's type assertions (`args["task_id"].(string)`). If the caller passes a non-string value, the assertion returns the zero value (`""`), and the tool returns an error. This is safe but provides poor error messages.

### Verdict

The MCP surface is completely unauthenticated. Any process can invoke any tool. The lifecycle is enforced at the repository level, so state machine bypass is not possible through MCP. But identity spoofing, unauthorized state changes, and misleading activity records are all possible.

---

## 12. HTTP API ADVERSARIAL AUDIT

### Authentication: NONE

As documented in Section 5, the `Auth` middleware exists but is never applied. All endpoints are open.

### Authorization: NONE

No endpoint checks caller identity, role, or permissions. Any caller can:
- Create/read/update/delete any project
- Create/read/update/claim/submit/accept/reject/block/unblock any task
- Post arbitrary activity on any task
- Query governance for any task

### Field-Level Mutation

The `UpdateTask` endpoint accepts `TaskUpdateFields` which includes `GovernanceRef`. An adversarial caller can:
1. Create a task with a legitimate governance_ref pointing to real Solvent
2. Later update the governance_ref to point to a fake provider
3. Query the fake provider via `GetGovernance`
4. Use the fake governance state to influence downstream decisions

This is a **shadow authority attack**: Conductor faithfully projects whatever the attacker's fake provider returns.

### Lifecycle Enforcement

Lifecycle transitions are enforced at the repository level. The `Transition` method rejects invalid state transitions. However, the `Update` method allows changing `title`, `description`, `priority`, and `governance_ref` on any task regardless of state. This is correct for non-lifecycle fields, but the `governance_ref` mutation is particularly dangerous (see above).

### Invalid Transitions

The `Transition` method correctly rejects invalid transitions (e.g., `proposed -> accepted`). This is tested.

### Cross-Project Access

`ListTasks` uses `TaskRepo.ListByProject(projectID)` which is correctly scoped. `GetTask` uses `TaskRepo.GetByID(taskID)` which does not validate project membership. A caller who knows a task ID can read it regardless of project boundaries. This is a **cross-project information leak**.

### Replay/Retry Behavior

The `Claim` method is idempotent-safe (conditional UPDATE). The `Transition` method is idempotent-safe (optimistic check on `fromStatus`). Retrying a failed claim or transition will not cause duplicate state changes. This is correct.

### Concurrency

As discussed in Section 4, the `Claim` and `Transition` methods use database-level atomicity. Concurrent claims and transitions are safe.

### Verdict

The HTTP API is a completely open coordination surface. Authentication is absent. Authorization is absent. Cross-project information leaks via direct task ID access. The `governance_ref` update path enables shadow authority attacks.

---

## 13. CROSS-PROJECT ISOLATION

### Dependency Isolation

The `DependencyRepository.Create` method correctly checks that both tasks belong to the same project:
```go
if taskProjectID != blockedByProjectID {
    return ErrSameProject
}
```

This is enforced at the database level (application check, not a DB constraint). An attacker cannot create cross-project dependencies.

### Task ID Enumeration

As noted in Section 12, `GetTask` does not validate project membership. A caller who knows a task ID from project A can read it even if they only have access to project B. This is a cross-project information leak.

### Project Deletion

`ProjectRepository.Delete` deletes the project but does not cascade to tasks or activities (relying on SQLite foreign keys with `ON DELETE CASCADE` would be safer, but the current schema does not specify `ON DELETE CASCADE`). Actually, looking at the schema:
```sql
project_id TEXT NOT NULL REFERENCES conductor_project(id)
```

Without `ON DELETE CASCADE`, deleting a project will fail if tasks reference it. This is actually **referential integrity protection**, which is good. But it means project deletion is effectively blocked by existing tasks, which may be unexpected behavior.

### Verdict

Cross-project dependency creation is correctly prevented. Cross-project task enumeration via `GetTask` is a leak. Project deletion is protected by referential integrity, which is correct but may be surprising.

---

## 14. DATA MODEL MINIMALITY

### Entities

| Entity | Purpose | Verdict |
|---|---|---|
| `Project` | Work container | Necessary, minimal |
| `Task` | Work item with lifecycle | Necessary, minimal |
| `Activity` | Append-only history | Necessary, minimal |
| `Dependency` | Task ordering | Necessary, minimal |

### Fields

| Field | Purpose | Verdict |
|---|---|---|
| `Task.current_agent` | Assignment tracking | Necessary |
| `Task.governance_ref` | Governance reference | Necessary, correctly opaque |
| `Activity.details` | Observational metadata | Necessary, correctly non-authoritative |
| `Activity.actor_type` / `actor_id` | Attribution | Necessary |

### Unused/Dead Code

- `GovernanceService` — provider routing logic, never instantiated
- `TransitionWithAgent` — combines transition + agent assignment, never used
- `TaskRepository.Release` — release a claimed task, never exposed
- `CheckAuthorization` on `GovernanceReader` — informational query, never called

### Creeping Concepts Absent

No agent registry, no workflow engine, no policy engine, no authorization state, no execution state, no domain ontology, no artifact store, no notification system, no event bus, no scheduler.

### Verdict

The data model is minimal and correct. Dead code exists but does not bloat the model.

---

## 15. FAILURE / RECOVERY AUDIT

### Agent Crash / Abandoned Task

If an agent crashes after claiming a task, the task remains in `active` status with `current_agent` set. There is no timeout or heartbeat mechanism. The task is permanently stuck until a human or another agent releases it — but `Release` is not exposed through any API or MCP surface. **Abandoned tasks cannot be recovered without direct database manipulation.**

### Database Failure

SQLite WAL mode is enabled. Foreign keys are enabled. Transaction rollback is correctly implemented. If the database file is corrupted or inaccessible, all operations fail. There is no replication or backup.

### Governance Failure

The Solvent adapter has a 10-second HTTP timeout. If Solvent is unreachable, `GetState` returns an error. The `GetGovernance` endpoint handles this by returning a state with `Status: unknown` and the error as a blocker. This is correct degradation.

### Malformed External Response

The Solvent client uses `json.NewDecoder` which will return an error for malformed JSON. This is handled correctly (error propagates to caller).

### UUID Collision Under Failure

If `generateUUID` produces a duplicate ID during activity creation, the INSERT fails. The transaction rolls back, so the state transition is also rolled back. The task remains in its previous state, and the activity is not recorded. This is correct failure behavior, but the collision risk itself is a bug.

### Verdict

Abandoned task recovery is impossible through any exposed surface. UUID collisions under concurrency are a real risk. Governance failures degrade safely.

---

## 16. DOMAIN-WORKLOAD TEST

The existing `domain_test.go` tests confirm Conductor works for restaurant reservations and library systems without core model changes. The `nextTask` issue (returning first proposed task without dependency check) is a coordination bug that would affect any domain with ordered dependencies, not a domain-specific flaw.

**Mental substitution tests:**
- **Go software development:** Works. Tasks = features/bugs. Dependencies = blocking PRs. Activities = commits/reviews.
- **Cybersecurity:** Works. Tasks = incident response steps. Dependencies = evidence chain. Governance = authorization for containment actions.
- **Scientific computing:** Works. Tasks = experiment runs. Dependencies = data prerequisites. Activities = results.
- **Data engineering:** Works. Tasks = pipeline steps. Dependencies = upstream tables. Governance = schema change authorization.
- **Infrastructure operations:** Works. Tasks = deploy steps. Dependencies = environment readiness. Governance = change approval.

The core model is sufficient for all these domains. The `nextTask` ordering issue would be a problem for domains requiring dependency-aware scheduling.

**Verdict:** PASS. Conductor coordinates arbitrary knowledge-intensive work without understanding domain meaning.

---

## 17. FOUR-WAY SEPARATION TEST

| Capability | Work | Authority | Execution |
|---|---|---|---|
| Agent has coding tools | Agent is assigned a task (`current_agent`) | Solvent decides if action is allowed | External executor runs the action |
| Agent can reason | Task lifecycle tracks progress | `CheckAuthorization` is informational | Conductor never executes |
| Agent produces output | Activity records what happened | `governance_ref` links to authority | Agent executes via external tools |
| Agent requests context | `next_task` finds work | `GetGovernance` observes authority | Solvent never executes |

### Separation Violations Found

1. **Activity != Execution fact:** Activity records like `work.completed` are observational, but nothing prevents an adversarial agent from recording `deploy.completed` or `execution.finished`. A downstream consumer could misread these as execution evidence.

2. **Work != Authority:** The `governance_ref` correctly separates work (Conductor) from authority (Solvent). However, since `GetState` always returns `unknown`, the authority plane is effectively disconnected. A reviewer seeing `unknown` might incorrectly infer "no authority required" or "authority not yet granted."

3. **Capability != Work:** An agent with coding tools is not automatically assigned work. Correct. But any agent can claim any unassigned task, regardless of capability. There is no capability matching.

4. **Authority != Execution:** Correctly preserved. Conductor never executes. Solvent never executes. External systems execute.

### Verdict

The four-way separation is correctly designed and mostly correctly implemented. The main risk is the **disconnected authority plane** (stub `GetState`) which makes it impossible to observe real authority state, and the **unauthenticated surfaces** which allow any caller to perform any work.

---

## SUMMARY OF CRITICAL FINDINGS

### P0 — Architectural Collapse (must fix before any production use)

1. **`Auth` middleware never applied** — All HTTP API endpoints are unauthenticated. `internal/api/routes.go` never calls `s.Auth` or any middleware chain.
2. **MCP zero authentication** — Any stdin writer can invoke any tool as the fixed `agentID`.
3. **Task creation bypasses claim** — `CreateTask` accepts `current_agent` in the JSON body, allowing pre-assignment.
4. **`GetState` is a non-functional stub** — Always returns `unknown`. Real Solvent query is unreachable through the public interface.
5. **`GovernanceService` is dead code** — Provider routing logic exists but is never wired into API or web servers.

### P1 — High Risk (significant architectural degradation)

6. **`CheckAuthorization` is dead code in the public interface** — Creates false impression that authorization is performed.
7. **Web UI has no authentication** — All project/task data exposed without identity check.
8. **Cross-project task enumeration** — `GetTask` does not validate project membership.
9. **`governance_ref` mutation enables shadow authority** — Any caller can redirect governance to a fake provider.
10. **`Release` method unreachable** — Abandoned tasks cannot be recovered through any API.

### P2 — Implementation Gaps (correctness risks)

11. **UUID collisions** — `generateUUID` is timestamp-based, not a UUID. Collisions possible under concurrency.
12. **`nextTask` ignores dependencies** — Returns first proposed task regardless of blocking dependencies.
13. **`Update` not transactional** — Multiple field updates are not atomic.
14. **Activity action strings uncontrolled** — Adversarial agents can record misleading action names.
15. **No capability matching** — Any agent can claim any task regardless of capability.

### P0/P1 Items That Break the Architecture Directly

Items 1, 2, 3, 4, 5, 6, 7, 8, 9 are the ones that either collapse the identity/authority boundary or make the Solvent integration non-functional. These are not "code quality" issues — they are **architectural boundary failures** that a malicious agent will exploit immediately.
