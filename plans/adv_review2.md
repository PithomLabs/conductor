# CONDUCTOR — SECOND ADVERSARIAL REVIEW

**Repository:** `/home/chaschel/Documents/go/conductor`
**Review Date:** 2026-09-11
**Reviewer Role:** Senior Go Architect / Adversarial Reviewer
**Mode:** Independent post-remediation differential review. No modifications performed.

---

## 1. EXECUTIVE VERDICT

**YELLOW — ADDITIONAL REMEDIATION REQUIRED**

The remediation addressed most structural defects from the first review. Authentication now guards the HTTP API surface. Task creation correctly rejects lifecycle bypass. The Solvent adapter queries real reference-specific state. `GovernanceService` is wired. `governance_ref` is write-once. `Release` is exposed. UUIDs use `google/uuid`.

However, the review found:

1. **CRITICAL lifecycle bypass in `Release`**: status is not validated. An agent can release a task from `review`, `blocked`, or `accepted` back to `proposed`, breaking the state machine.
2. **Web UI is completely unauthenticated**: all project/task/governance data is exposed without identity.
3. **MCP is completely unauthenticated**: any process with stdin access can invoke any tool as the fixed `agentID`.
4. **Inconsistent governance error handling**: HTTP returns `unknown`, MCP returns error for identical Solvent failures.
5. **Cross-project task enumeration** remains unaddressed.
6. **Dead exported code** (`compareKeys`) creates future risk.

These are not test failures. Tests pass because they do not attack the uncovered paths.

---

## 2. DIFFERENTIAL FINDINGS F1–F11

| Finding | Original Defect | Remediation Status | Regression | HTTP | MCP | Repository | Verdict |
|---|---|---|---|---|---|---|---|
| F1 Auth middleware dead | `Auth` defined but never wired | **Partially fixed.** HTTP API now wraps `/v1/` with `NewAuthMiddleware`. `ListenAndServe` constructs auth chain. | None | Wrapped | Not wrapped | N/A | **PARTIAL** |
| F2 MCP zero authentication | No identity on stdio server | **Not fixed.** MCP `agentID` is fixed at construction. No per-request auth. | None | N/A | Zero auth | N/A | **OPEN** |
| F3 Task creation bypass | `CreateTask` accepted `current_agent` and arbitrary `status` | **Fixed.** `CreateTask` handler rejects non-proposed status and non-nil `current_agent`. `TaskRepository.Create` enforces same invariants and returns `ErrLifecycleBypass`. | None | Enforced | Enforced (via repo) | Enforced | **FIXED** |
| F4 GetState stub | Always returned `unknown` | **Fixed.** `Adapter.GetState` now requires `belief_id` in metadata and calls `client.GetBeliefExplain`. | None | Uses adapter | Uses adapter | Uses adapter | **FIXED** |
| F5 GovernanceService dead | Never instantiated | **Fixed.** `main.go` creates `GovernanceService`, registers `null` and optionally `solvent`, passes to API, web, and MCP. | None | Wired | Wired | Wired | **FIXED** |
| F6 CheckAuthorization misleading | Dead code in public interface | **Partially fixed.** Documented as reserved in `reader.go` comments. Still present in `GovernanceReader` interface and `GovernanceService`. Not surfaced in HTTP/MCP. | None | Not exposed | Not exposed | Present but unused | **PARTIAL** |
| F7 Web UI no authentication | All data exposed | **Not fixed.** `web.NewServer` and `web.Server` have no auth. All handlers are open. | None | N/A | N/A | N/A | **OPEN** |
| F8 Cross-project GetTask | No project membership validation | **Not fixed.** `GetTask` handler and MCP `getTask` call `TaskRepo.GetByID` without project check. | None | Leaks | Leaks | Leaks | **OPEN** |
| F9 governance_ref mutation | `TaskUpdateFields` included `GovernanceRef` | **Fixed.** `TaskUpdateFields` struct removed `GovernanceRef`. Only `Title`, `Description`, `Priority` remain. | None | Blocked | Blocked | Blocked | **FIXED** |
| F10 Release unreachable | `Release` not exposed | **Fixed.** `ReleaseTask` HTTP handler and `conductor_release_task` MCP tool added. | **New bypass found** (see §9) | Wired | Wired | Exposed | **FIXED with regression** |
| F11 UUID collisions | Timestamp-based fake UUID | **Fixed.** `generateUUID` now uses `github.com/google/uuid`. | None | N/A | N/A | Fixed | **FIXED** |

---

## 3. NEW FINDINGS

### CRITICAL

**C1: `Release` bypasses status validation (lifecycle state machine break)**

- **Location**: `internal/store/task_repo.go` lines 277–313
- **Observed behavior**: `Release` reads `current_agent` and `status` from the database but only validates `current_agent`. It never checks `status`. The UPDATE unconditionally sets `status = 'proposed'`.
- **Exploit scenario**:
  1. Agent A claims task → `active`, `current_agent = A`
  2. Agent A submits → `review`
  3. Reviewer accepts → `accepted`
  4. Agent A calls `Release` → `proposed`, `current_agent = NULL`
  5. Task is back to proposed despite being accepted. The acceptance is silently undone.
- **Impact**: The state machine is not enforced. `accepted` and `review` are not protected states. Any claimed task can be reset to `proposed` by its assignee at any time.
- **Recommended remediation**: Add status check in `Release`:
  ```go
  if currentStatus != domain.TaskStatusActive {
      return ErrInvalidTransition
  }
  ```
- **Architecture change required**: NO — this is a bug fix in the existing state machine enforcement.

**C2: Web UI has no authentication (complete data exposure)**

- **Location**: `internal/web/server.go` — all handlers (`handleDashboard`, `handleProject`, `handleTask`) call repositories directly without identity checks.
- **Observed behavior**: Any network client can read all projects, tasks, activities, and governance state via the web UI. No API key required.
- **Exploit scenario**: An attacker with network access to the web port can enumerate all projects, read task details, see agent assignments, and observe governance state.
- **Impact**: Complete confidentiality breach of coordination data.
- **Recommended remediation**: Either apply the same `Auth` middleware to web routes, or document the web UI as trusted-local-only and bind it to `127.0.0.1` by default.
- **Architecture change required**: NO — the v1 scope says "single trusted coordination workspace"; the UI should be protected by the same auth boundary as the API.

### HIGH

**H1: MCP has zero authentication (unauthenticated coordination surface)**

- **Location**: `internal/mcp/server.go`, `cmd/conductor/main.go` lines 66–77
- **Observed behavior**: MCP `agentID` is fixed at construction. No per-request identity verification. Any process that can write to the MCP server's stdin can invoke any tool.
- **Exploit scenario**: A malicious agent or compromised process invokes `conductor_claim_task`, `conductor_submit_task`, `conductor_accept_task`, `conductor_release_task`, etc., as the fixed `agentID`.
- **Impact**: Full coordination control without identity.
- **Recommended remediation**: For stdio-local deployment, document the trust boundary. If remote MCP exposure is ever added, add per-request auth. For now, the risk is operational (compromised local process), not network-remote.
- **Architecture change required**: NO — but the trust boundary must be explicitly documented.

**H2: Inconsistent governance error handling between HTTP and MCP**

- **Location**: `internal/api/governance_handler.go` lines 50–57 vs `internal/mcp/tools.go` lines 305–309
- **Observed behavior**:
  - HTTP `GetGovernance`: on `GetState` error, returns `status: unknown` (swallows error).
  - MCP `getGovernance`: on `GetState` error, returns `MCPResponse{Error: err.Error()}` (propagates error).
- **Exploit scenario**: A client cannot predict behavior. An agent using MCP sees failures; an agent using HTTP sees `unknown`. The same underlying state produces different responses.
- **Impact**: Observability confusion. An agent might interpret HTTP `unknown` as "no governance" while MCP reports a hard error.
- **Recommended remediation**: Align both paths. Either propagate errors in both or return `unknown` in both. HTTP currently documents itself as "read-only observation" so returning `unknown` on error is defensible; MCP should match.
- **Architecture change required**: NO.

**H3: Cross-project task enumeration via `GetTask`**

- **Location**: `internal/api/task_handler.go` lines 31–50, `internal/mcp/tools.go` lines 79–90
- **Observed behavior**: `GetTask` and `getTask` call `TaskRepo.GetByID(taskID)` without validating that the task belongs to a project the caller can access.
- **Exploit scenario**: An authenticated caller who observes a task ID from another project (e.g., via activity log, dependency, or ID prediction) can read its full details.
- **Impact**: Cross-project information leak. In v1 single-workspace model this is partially mitigated by "trusted callers," but task IDs are predictable (UUIDs) and may appear in logs or URLs.
- **Recommended remediation**: Either add project membership check to `GetTask`/`getTask`, or document that v1 assumes fully trusted callers and task IDs are not secret.
- **Architecture change required**: NO — but scope must be explicit.

### MEDIUM

**M1: Dead exported code `compareKeys`**

- **Location**: `internal/api/auth.go` lines 58–61
- **Observed behavior**: `compareKeys` is exported but never called. The `Auth` middleware uses map lookup, not constant-time comparison.
- **Impact**: Misleading. A future developer might use `compareKeys` for credential comparison, but it compares the wrong values (API key vs API key, not stored hash). Also, map lookup is not constant-time.
- **Recommended remediation**: Remove `compareKeys` or use it correctly with stored hashes. For v1 POC, map lookup is acceptable.
- **Architecture change required**: NO.

**M2: MCP `agentID` derivation in `main.go` is non-deterministic**

- **Location**: `cmd/conductor/main.go` lines 66–72
- **Observed behavior**: In MCP mode, `agentID` is set to the first value from the credentials map. Go map iteration order is randomized. The same credentials produce different `agentID` values across restarts.
- **Impact**: Activity records and task assignments in MCP mode have non-deterministic actor identity.
- **Recommended remediation**: Use a deterministic mapping (e.g., hash of the key) or require explicit `--agent-id` flag.
- **Architecture change required**: NO.

**M3: `Release` does not validate terminal states**

- **Location**: `internal/store/task_repo.go` lines 277–313 (related to C1)
- **Observed behavior**: Even if status check is added, `Release` should also reject terminal states (`accepted`, `cancelled`). Current code would allow releasing a cancelled task if it had `current_agent` set.
- **Impact**: A cancelled task could be resurrected.
- **Recommended remediation**: Check `currentStatus == domain.TaskStatusActive` before releasing.
- **Architecture change required**: NO.

### LOW

**L1: Web UI template ignores governance errors silently**

- **Location**: `internal/web/server.go` lines 140–147
- **Observed behavior**: `handleTask` ignores error from `s.Governance.GetState`. If Solvent query fails, governance panel is absent. HTTP `GetGovernance` returns `unknown` on error.
- **Impact**: Inconsistent observability between web UI and API.
- **Recommended remediation**: Pass error state to template or log it.
- **Architecture change required**: NO.

**L2: `CreateTask` HTTP handler allows `governance_ref` at creation without validation**

- **Location**: `internal/api/task_handler.go` lines 61–76
- **Observed behavior**: `CreateTask` does not strip or validate `governance_ref`. The repo `Create` method stores whatever is provided. This is by design (write-once at creation), but the JSON is not validated for structural correctness.
- **Impact**: A caller can set `governance_ref` to malformed JSON or a reference pointing to a non-existent provider. The error only surfaces later on `GetGovernance`.
- **Recommended remediation**: Validate `governance_ref` is valid JSON with required fields, or accept it as opaque and let `GetGovernance` handle errors.
- **Architecture change required**: NO.

---

## 4. BOUNDARY MATRIX

| Responsibility | Intended Owner | Actual Owner | Status |
|---|---|---|---|
| Project lifecycle | Conductor | Conductor | OK |
| Task lifecycle / state machine | Conductor | Conductor | OK (with C1 bypass) |
| Task assignment (`current_agent`) | Conductor | Conductor | OK |
| Task dependency ordering | Conductor | Conductor | OK |
| Activity / work history | Conductor | Conductor | OK (append-only) |
| Governance observation | Conductor (read-only) | Conductor | OK (with H2 inconsistency) |
| Governance authority | Solvent | Solvent | OK |
| Authorization decision | Solvent | Solvent | OK |
| Consequential execution | External executor | External executor | OK |
| Authentication (HTTP API) | Conductor | Conductor | OK (web/MCP gaps) |
| Authorization to coordination ops | Conductor | Conductor | OK (no RBAC in v1) |
| Domain semantics | Domain application | Not in Conductor | OK |

---

## 5. FOUR-WAY SEPARATION MATRIX

| Capability | Work | Authority | Execution | Correct? |
|---|---|---|---|---|
| Agent identity (API key) | Authentication | N/A | N/A | YES |
| Task creation | Work coordination | N/A | N/A | YES |
| Task claim | Work assignment | N/A | N/A | YES |
| Task submit/review/accept | Work progression | N/A | N/A | YES |
| Task release | Work reassignment | N/A | N/A | YES (with C1 status bypass) |
| Governance query | Observation | Solvent authority | N/A | YES |
| `CheckAuthorization` | N/A | Informational only | N/A | YES (reserved) |
| Activity recording | Observational history | N/A | N/A | YES |
| Solvent `GetState` | N/A | Authority state | N/A | YES |
| Solvent `VerifyAuthorization` | N/A | Authority decision | N/A | YES |

**Key separation violations:**

- `Release` currently infers work-state reset (`proposed`) from a work action (agent request), bypassing the state machine. It should require the work to be in `active` state, not just assigned.
- `governance_ref` is correctly non-authoritative, but the web UI's silent error handling on governance queries can make `unknown` look like "no governance" rather than "governance unavailable."

---

## 6. SECURITY FINDINGS

### Authentication
- **HTTP API**: `NewAuthMiddleware` wraps all `/v1/` routes. Missing/invalid API key returns 401. Credentials loaded from `CONDUCTOR_API_KEY` env var. Format: `key1:actor-1,key2:actor-2`.
- **Gap**: Web UI has no auth. MCP has no auth.
- **Gap**: `compareKeys` is exported but unused; actual credential comparison is map lookup (not constant-time, but acceptable for POC).

### Actor Identity
- HTTP: `agentIDFromContext` returns authenticated actor ID. Cannot be spoofed by request body.
- MCP: `agentID` is fixed at construction. Caller-controlled via stdin.
- Activity: `ActorID` is set from authenticated context (HTTP) or fixed `agentID` (MCP). Not spoofable via request body.

### Lifecycle Bypass
- **C1 (CRITICAL)**: `Release` does not check status. Accepted/reviewed/blocked tasks can be reset to `proposed`.

### Concurrency
- `Claim` uses conditional UPDATE — safe.
- `Transition` uses optimistic status check inside transaction — safe for SQLite.
- `Release` uses conditional UPDATE on `current_agent` but not on `status` — allows concurrent release of non-active tasks.

### Governance Ref
- Write-once at creation. Cannot be mutated via `Update`. Good.
- Content is opaque JSON. Not interpreted by Conductor. Good.
- Solvent adapter requires `belief_id` in metadata. Good.

### Solvent Integration
- Production path: HTTP/MCP → `GovernanceService.GetState` → `SolventAdapter.GetState` → `client.GetBeliefExplain` → Solvent. Correct.
- NullReader used when no `SOLVENT_URL` configured. Correct.
- Unknown provider returns `unknown` with blocker. Correct.
- Errors propagate as `unknown` in HTTP, as error in MCP. Inconsistent (H2).

### Resource Access
- Single-workspace v1 model documented in `governance/types.go`.
- `GetTask` and `getTask` do not validate project membership. Cross-project enumeration possible.

### Activity Semantics
- Append-only. No UPDATE/DELETE paths.
- Action strings are uncontrolled. Any string accepted.
- Activity is observational; no consumer interprets actions as authority or execution.

---

## 7. SOLVENT BOUNDARY

| Question | Answer | Evidence |
|---|---|---|
| Can Conductor authorize? | NO | No authorization endpoint. `CheckAuthorization` is reserved/informational only. |
| Can Conductor execute? | NO | No execution paths. No shell, no command runner, no external mutation. |
| Can Conductor mutate Solvent? | NO | Adapter is read-only. Only `GetState` and `CheckAuthorization` (informational). |
| Can Conductor create competing authority state? | NO | `governance_ref` is opaque reference. Conductor never stores Solvent-derived truth locally. |
| Can Conductor infer authority from work state? | NO | `current_agent` and task status are work state. Not used for authorization. |
| Can Solvent manage Conductor task lifecycle? | NO | Solvent has no knowledge of Conductor tasks. No reverse integration. |
| Can Solvent manage project state? | NO | Same as above. |
| Can Solvent assign agents? | NO | `current_agent` is set only by Conductor `Claim` or `Release`. |

---

## 8. DOMAIN-AGNOSTICITY

The core model (Project, Task, Activity, Dependency) has no domain-specific fields, status meanings, or lifecycle semantics. `next_task` uses generic dependency resolution. `governance_ref` is opaque JSON.

Mental substitution tests:
- **Go development**: Tasks = features/bugs. Dependencies = PR blocks. Governance = deploy authorization.
- **Cybersecurity**: Tasks = incident steps. Dependencies = evidence prerequisites. Governance = containment authorization.
- **Scientific computing**: Tasks = experiment runs. Dependencies = data availability. Governance = protocol authorization.
- **Data engineering**: Tasks = pipeline stages. Dependencies = upstream tables. Governance = schema change approval.
- **Infrastructure**: Tasks = deploy steps. Dependencies = environment readiness. Governance = change approval.

**Verdict**: PASS. Conductor remains domain-agnostic.

---

## 9. MINIMALITY

No new abstractions were introduced that violate minimality. The `GovernanceService` provider-routing layer was pre-existing dead code that is now correctly wired — this is recovery, not scope creep.

Dead code introduced or left behind:
- `compareKeys` (exported, unused)
- `TransitionWithAgent` (unused)
- `CheckAuthorization` on `GovernanceReader` (reserved, unused)

None of these bloat the model. They are future-risk items, not active complexity.

---

## 10. FINAL DECISION

**YELLOW — ADDITIONAL REMEDIATION REQUIRED**

The architecture is substantially stronger than the first review. The execution boundary is intact. The Solvent integration is functional. The data model is minimal. Domain-agnosticity is preserved.

However:

1. **C1 (CRITICAL)**: `Release` must validate task status. This is a lifecycle state machine bypass that undermines the `accepted` terminal state.
2. **C2 (CRITICAL)**: Web UI must be authenticated or explicitly scoped as trusted-local.
3. **H1 (HIGH)**: MCP authentication must be documented as a trust-boundary decision. If MCP is ever exposed beyond local stdio, auth is required.
4. **H2 (HIGH)**: Governance error handling must be consistent between HTTP and MCP.
5. **H3 (HIGH)**: Cross-project `GetTask` must be addressed in v1 scope documentation.

Until C1 is fixed, the lifecycle cannot be considered enforced. A malicious agent with valid low-privilege credentials can undo accepted work via `Release`.

---

## APPENDIX: VERIFIED REMEDIATION DETAILS

### Auth Middleware (F1)
- `internal/api/auth.go`: `NewAuthMiddleware` takes `map[string]string`, validates API key, derives actor ID.
- `internal/api/server.go`: `ListenAndServe` creates `authMux.Handle("/v1/", NewAuthMiddleware(s.Credentials)(mux))`.
- Tests use `wrappedMux` helper that applies auth.
- **Gap**: Web UI and MCP are not wrapped.

### Task Creation Bypass (F3)
- `internal/api/task_handler.go`: `CreateTask` rejects `task.Status != "" && task.Status != "proposed"` and `task.CurrentAgent != nil`.
- `internal/store/task_repo.go`: `Create` enforces same checks and returns `ErrLifecycleBypass`.
- Tests cover: non-proposed status, non-nil current_agent, empty status, proposed status accepted.

### Solvent GetState (F4)
- `internal/adapter/solvent/adapter.go`: `GetState` requires `ref.ReferenceID != ""` and `ref.Metadata["belief_id"]` (non-empty string). Calls `client.GetBeliefExplain`.
- No longer returns stub `unknown`.

### GovernanceService (F5)
- `cmd/conductor/main.go`: Creates `service.NewGovernanceService()`, registers `null` and optionally `solvent`.
- Passed to `api.NewServer`, `web.NewServer`, `mcp.NewServer`.

### CheckAuthorization (F6)
- `internal/governance/reader.go`: Comment documents it as "reserved for future governance capabilities; not surfaced in V1 HTTP or MCP interfaces."
- Still in interface but not exposed.

### governance_ref Write-Once (F9)
- `internal/store/task_repo.go`: `TaskUpdateFields` has only `Title`, `Description`, `Priority`. No `GovernanceRef`.
- `CreateTask` does not strip `governance_ref` from request, but it is write-once because `Update` cannot modify it.

### Release Exposed (F10)
- HTTP: `ReleaseTask` at `/v1/tasks/{id}/release`.
- MCP: `conductor_release_task`.
- Repo: `Release` checks `current_agent` matches caller, atomically clears `current_agent` and sets `status = 'proposed'`, records `task.released` activity.
- **Bypass**: Does not check `status` before releasing.

### UUID Generation (F11)
- `internal/store/task_repo.go`: `generateUUID` uses `github.com/google/uuid`.
- MCP `tools.go`: `postActivity` uses `uuid.New().String()`.
