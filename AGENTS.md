# AGENTS.md — Conductor Engineering Contract

This is the durable repository-level engineering contract for AI coding agents
and human contributors. The codebase is the source of truth; if this document
conflicts with the code, the code wins and this document must be updated.

---

## 1. What Is Conductor

Conductor is a coordination substrate for agent-facing project and work
management. It is domain-agnostic: the same core serves software engineering,
scientific work, cybersecurity, data engineering, infrastructure operations,
and other knowledge-intensive domains.

Conductor answers:
- What work exists?
- Who is doing it?
- What depends on what?
- What is the current work state?
- What happened in the project workflow?

Conductor does NOT answer:
- Is a domain proposition true?
- Is a scientific claim valid?
- Is a consequential action authorized?
- Did an external consequence actually occur?
- What does a domain artifact mean?

---

## 2. The Four Actors

| Role | Name | Owns |
|------|------|------|
| Agency | Agent | Reasoning, interpretation, investigation, implementation, tool use, artifact production, ordinary work, requesting consequential actions |
| Coordination | Conductor | Project/work state, tasks, dependencies, assignments, lifecycle, activity, agent-facing API/MCP, human UI, read-only governance observation |
| Authority | Solvent | Authority, authorization, exact target/state binding, intent, revocation, claim, consequential authorization boundary |
| Effect | Executor | Actual external effects, actual external execution outcomes |

These four roles are non-overlapping. Every authoritative fact has one owner.

---

## 3. Hard Invariants

```
CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION
```

- Agent capability does NOT imply work assignment.
- Work assignment does NOT imply authority.
- Authority does NOT imply execution.
- Execution does NOT imply authority.

Conductor enforces this by:
- Never executing work.
- Never authorizing consequences.
- Never inferring authority from task state.
- Never inferring execution from Activity records.

```
DOMAIN MEANING is outside the Conductor infrastructure layer.
```

---

## 4. What Conductor Owns

Conductor owns:
- Projects (CRUD, status lifecycle)
- Tasks (CRUD, lifecycle state machine)
- Dependencies (same-project blocking relationships)
- Assignments (claim/release with atomic SQL)
- Lifecycle transitions (validated at repository layer)
- Activity (append-only event log)
- Authentication boundary for access to the coordination interface
  (API key → stable actor identity; authentication ≠ authorization
  ≠ Solvent authority)
- Governance reference: Conductor-owned opaque linkage to an external
  governance system. Conductor owns the reference, not the governance
  authority represented by the reference.
- Read-only governance observation (via GovernanceReader interface)
- MCP agent interface (15 tools, stdio JSON-RPC)
- Minimal human web UI (Kanban board, task detail, governance panel)

---

## 5. What Conductor Must NOT Own

Conductor MUST NOT own:
- Authority or authorization decisions
- Execution brokering or effect management
- Domain truth or domain semantics
- Shadow authorization state
- Shadow execution state
- Duplicate Solvent authority state
- Agent runtime, scheduling, or execution features
- Workflow engine features (complex DAG execution, conditional branching)
- Policy engine features (rule evaluation, constraint solving)
- Research engine features (domain-specific task states, evidence tracking)
- Project-management features merely because conventional PM tools have them

---

## 6. Non-Overlap Rule

Solvent MUST NOT own:
- Projects, tasks, dependencies, assignment
- Conductor lifecycle or project activity

Domain applications/workloads MUST NOT require Conductor changes merely
because they have different domain semantics.

---

## 7. Agent Interaction Model

Agents interact with Conductor exclusively through:
- **API**: HTTP under `/v1/` with authentication
- **MCP**: stdio JSON-RPC with 15 tools

Agents MUST NOT access repositories or the database directly.

Agent identity derives from the authenticated boundary:
- API: API key maps to stable actor ID via `CONDUCTOR_API_KEY`
- MCP: `CONDUCTOR_MCP_AGENT_ID` env var or default `"agent-mcp"`

Agents are treated as potentially buggy or adversarial callers. Conductor
does not trust agent claims about task state, authority, or execution
outcomes.

Standard agent workflow:
1. Discover available tasks (`conductor_next_task` or `conductor_list_tasks`)
2. Claim a task (`conductor_claim_task`)
3. Perform work (external to Conductor)
4. Record activity (`conductor_post_activity`)
5. Submit for review (`conductor_submit_task`)

---

## 8. Solvent Integration Boundary

Conductor observes external governance through a single read-only interface:

```go
type GovernanceReader interface {
    GetState(ctx context.Context, ref GovernanceReference) (*GovernanceState, error)
}
```

`CheckAuthorization` is intentionally absent from this interface.
Authorization goes directly Agent → Solvent, not through Conductor.

Rules:
- The governance reference is provider-routable; provider-specific payload is opaque to Conductor.
- Solvent unavailability produces `GovernanceState{status: "unknown", blockers: ["provider unavailable"]}`, not an error.
- Unknown/unavailable MUST NOT become authorization.
- External governance MUST NEVER become a required dependency of ordinary Conductor task coordination. Work coordination remains functional when governance is unavailable.
- Conductor MUST NEVER mutate Solvent state.
- Conductor MUST NEVER persist Solvent authorization as Conductor truth.
- Conductor MUST NEVER interpret provider-specific governance semantics in the generic core.

---

## 9. External Executor Boundary

Actual external execution happens outside Conductor. External executors own
actual external effects and execution outcomes.

Activity records are project/work history, NOT proof of external execution.

Conductor MUST NEVER become an execution broker.

---

## 10. Domain-Agnosticity Rules

Conductor MUST NOT encode domain-specific concepts:
- No physics, mathematics, software development, cybersecurity
- No scientific workflows, experiments, research ontology
- No proofs, hypotheses, evidence semantics
- No domain-specific policy or task states

Conductor MUST use only generic concepts:
- task, artifact, dependency, external action, governed consequence, project work

The same Conductor core MUST remain usable across unrelated domains without
domain-specific core changes. This has been validated by tests covering
restaurant reservations, library systems, and generic domains
(`internal/agentstub/domain_test.go`).

---

## 11. Data Model

### Entities

| Entity | Description |
|--------|-------------|
| Project | Top-level container. Status: active, completed, archived |
| Task | Unit of work. Assigned to an agent through lifecycle |
| Activity | Append-only event log entry. Records all state transitions |
| Dependency | Task-to-task blocking relationship. Same-project enforced |

### Task Lifecycle State Machine

```
proposed  ──claim──>   active
proposed  ──block──>   blocked
proposed  ──cancel──>  cancelled

active    ──submit──>  review
active    ──block──>   blocked
active    ──cancel──>  cancelled
active    ──release──> proposed

review    ──accept──>  accepted   (terminal)
review    ──reject──>  active
review    ──cancel──>  cancelled

blocked   ──resolve──> active
blocked   ──cancel──>  cancelled

accepted  (terminal — no outgoing transitions)
cancelled (terminal — no outgoing transitions)
```

### Invariants

- Tasks enter `proposed` state at creation with `current_agent = nil`.
- `status` and `current_agent` cannot be set at creation (enforced by
  `ErrLifecycleBypass`).
- Generic PATCH (`TaskUpdateFields`) can only change `title`, `description`,
  `priority`. Never `status`, `current_agent`, or `governance_ref`.
- `governance_ref` is write-once at creation, immutable after. Conductor
  owns the reference linkage, not the governance authority. Governance
  authority is owned exclusively by the external governance system.
- Claim is atomic: `WHERE status='proposed' AND current_agent IS NULL`.
- Release is atomic: `WHERE status='active' AND current_agent = ?`.
- Lifecycle state changes and Activity records are in the same transaction.
- Activity is append-only (never updated or deleted).

---

## 12. Authentication and Identity

- All `/v1/` API routes require authentication.
- Credentials map API key → stable actor identity. The API key is not the
  identity; it authenticates to reveal the identity.
- Conductor owns the authentication boundary for access to its coordination
  interface. Authentication ≠ authorization ≠ Solvent authority.
- MCP agent identity is deterministic: `CONDUCTOR_MCP_AGENT_ID` env var
  or fixed default `"agent-mcp"`. Never derived from credential map
  iteration.
- V1 is a single trusted workspace. Authentication establishes WHO is
  calling, not per-project authorization.
- Resource IDs (UUIDs) reduce enumeration risk but are NOT authorization
  boundaries.

MUST NOT add without explicit architectural review:
- RBAC
- Multi-tenancy
- Per-project authorization
- Project membership

---

## 13. API Surface

All routes under `/v1/`, all require authentication.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/v1/projects` | List projects |
| POST | `/v1/projects` | Create project |
| GET | `/v1/projects/{id}` | Get project |
| PUT | `/v1/projects/{id}` | Update project |
| DELETE | `/v1/projects/{id}` | Delete project |
| GET | `/v1/projects/{id}/tasks` | List tasks in project |
| POST | `/v1/projects/{id}/tasks` | Create task in project |
| GET | `/v1/tasks/{id}` | Get task |
| PATCH | `/v1/tasks/{id}` | Update task (non-lifecycle fields only) |
| POST | `/v1/tasks/{id}/claim` | Claim task |
| POST | `/v1/tasks/{id}/submit` | Submit for review |
| POST | `/v1/tasks/{id}/accept` | Accept work |
| POST | `/v1/tasks/{id}/reject` | Reject work |
| POST | `/v1/tasks/{id}/blocker` | Report blocker |
| POST | `/v1/tasks/{id}/resolve` | Resolve blocker |
| POST | `/v1/tasks/{id}/release` | Release task |
| POST | `/v1/tasks/{id}/activity` | Record activity |
| GET | `/v1/tasks/{id}/governance` | Read governance state (read-only) |

Governance observation endpoints are read-only and MUST expose only GET
semantics.

---

## 14. MCP Surface

15 tools via stdio JSON-RPC:

| Tool | Purpose |
|------|---------|
| `conductor_get_project` | Read project context |
| `conductor_list_tasks` | List tasks in project |
| `conductor_get_task` | Read specific task |
| `conductor_create_task` | Create new task |
| `conductor_update_task` | Update task (non-lifecycle fields) |
| `conductor_claim_task` | Claim task (atomic) |
| `conductor_submit_task` | Submit for review |
| `conductor_accept_task` | Accept work |
| `conductor_reject_task` | Reject work |
| `conductor_report_blocker` | Report blocking issue |
| `conductor_resolve_blocker` | Resolve blocking issue |
| `conductor_release_task` | Release claimed task |
| `conductor_post_activity` | Record activity |
| `conductor_get_governance` | Get governance status (read-only) |
| `conductor_next_task` | Get next unblocked task |

`next_task` returns the first proposed, unassigned task whose dependencies
are all resolved (blocker must be accepted or cancelled).

Governance failure returns `GovernanceState{status: "unknown",
blockers: ["provider unavailable"]}`, matching HTTP behavior.

---

## 15. Database Rules

- SQLite via pure Go driver (`modernc.org/sqlite`), WAL mode, foreign keys.
- Lifecycle state changes use `db.Transaction()` with auto-rollback.
- Claim and Release use atomic SQL with `RowsAffected` checks.
- Dependency creation enforces same-project constraint (`ErrSameProject`).
- All repositories use parameterized queries.

Error variables (defined in `internal/store/sqlite.go`):
- `ErrNotFound` — record not found
- `ErrInvalidTransition` — invalid lifecycle transition
- `ErrClaimFailed` — claim failed (already claimed or not proposed)
- `ErrSameProject` — dependency crosses project boundary
- `ErrLifecycleBypass` — generic update tries to mutate lifecycle fields
- `ErrReleaseFailed` — release from non-active state or wrong agent

The full test suite MUST pass before changes are committed.

---

## 16. Adapter and Integration Rules

External systems belong behind adapters/services, not embedded in the
generic domain model.

The GovernanceService routes to the correct adapter by provider name.
Unknown providers return `GovernanceState{status: "unknown"}` with
appropriate blocker, not an error.

The Solvent adapter (`internal/adapter/solvent/`) is the reference
implementation of the adapter pattern. It translates Solvent-specific
responses to generic governance types. Solvent-specific concepts
(e.g., `belief_id`) stay in the adapter, never in the generic core.

---

## 17. Web UI

The web UI MUST bind to loopback by default. The web UI MUST NOT be
exposed beyond loopback unless the deployment provides an explicit
authenticated access boundary.

No web authentication system in V1. Renders: project list, Kanban board
(tasks by status), task detail with activity log and governance panel.

---

## 18. Testing Expectations

- All lifecycle transitions have test coverage.
- Release-from-non-active-state is tested for every non-active status.
- Concurrent claim protection is tested.
- Governance provider failure semantics are tested (MCP and HTTP return
  same shape).
- Domain-agnosticity is tested via `internal/agentstub/domain_test.go`.
- `rtk go build ./...` and `rtk go test ./...` must pass before any commit.

---

## 19. Architectural Growth Gate

A new Conductor core primitive requires evidence that:
1. The requirement is a generic coordination invariant.
2. It cannot be expressed through the existing coordination model.
3. It cannot be expressed through an external adapter or service.

The fact that one domain workload needs a feature is NOT sufficient.

The fact that conventional PM tools have a feature is NOT sufficient.

Decision test: "Could this be implemented as an adapter, integration,
service, policy, or deployment boundary instead of a core change?"

Any proposed Solvent change must remain subject to the separate Solvent
kernel-growth discipline.

---

## 20. Anti-Patterns

Do NOT turn Conductor into:
- **Jira/Linear** — by adding project-management features without core
  justification
- **Agent framework** — by adding agent runtime, scheduling, or execution
  features
- **Workflow engine** — by adding complex DAG execution, conditional
  branching, or parallel fork/join
- **Research engine** — by adding domain-specific task states, evidence
  tracking, or proof management
- **Policy engine** — by adding rule evaluation, constraint solving, or
  authorization logic
- **Execution engine** — by adding work execution, effect brokering, or
  consequence management
- **Authorization system** — by adding RBAC, capability matching, or
  per-project authorization

Do NOT:
- Smuggle domain semantics into the generic core under the guise of
  "flexibility" or "extensibility"
- Persist Solvent authorization as Conductor truth
- Infer authorization from task state
- Infer execution from Activity records
- Trust agent claims without verification
- Allow generic PATCH to bypass lifecycle
- Set identity from arbitrary request metadata
- Expose the web UI on all interfaces without authentication
- Make ordinary coordination fail when Solvent is unavailable

---

## 21. Definition of Done

A Conductor implementation is done when:
- The generic coordination model serves the workload without domain-specific
  core changes.
- All lifecycle invariants are enforced at the repository layer.
- Governance observation is read-only and non-authoritative.
- No execution path exists through Conductor.
- No domain-specific leakage exists in the core.
- Tests pass and cover the critical invariants.
- AGENTS.md is updated if architectural decisions changed.

---

## 22. Repository-Specific Guidance

Before writing code:
1. Inspect the actual repository — do not assume documentation is more
   authoritative than code.
2. Check `go.mod` for actual dependencies before assuming a library is
   available.
3. Check existing code patterns before introducing new ones.
4. Read `internal/store/sqlite.go` for error variables.
5. Read `internal/domain/task.go` for the lifecycle state machine.
6. Read `internal/api/routes.go` for the API surface.
7. Read `internal/mcp/tools.go` for the MCP surface.
8. Read `internal/governance/reader.go` for the GovernanceReader interface.

After making changes:
1. `rtk go build ./...`
2. `rtk go test ./...`
3. Verify no new domain-specific concepts leaked into the core.

---

## 23. Appendix: Repository State

- **Module**: `github.com/PithomLabs/conductor`
- **Database**: SQLite via pure Go driver (`modernc.org/sqlite`)
- **Core tables**: 4 (project, task, activity, dependency)
- **Modes**: api (HTTP + auth), mcp (stdio JSON-RPC), web (HTML)
- **GovernanceReader interface**: single method `GetState`
- **Dependencies**: `github.com/google/uuid`, `modernc.org/sqlite`
