# Conductor — Implementation Plan (Revision 2)

## Architectural Decisions (Retained)

- **Conductor is a standalone project layer**
- **Separate repository and separate binary/process from Solvent**
- **Standalone SQLite database** (not Solvent's CockroachDB)
- **Never directly read or write Solvent's database**
- **Never become a second authority engine**
- **Solvent remains authoritative** for governance/authorization state
- **Conductor remains authoritative** for project/work state
- **Domain-agnostic**: no domain-specific concepts in core model
- **Generic governance boundary**: Solvent is one provider implementation

```text
                 CONDUCTOR
        ┌─────────────────────────┐
        │ Project lifecycle       │
        │ Task lifecycle          │
        │ Agent coordination      │
        │ Activity history        │
        │ Governance projection   │
        │ MCP + API               │
        │ Minimal UI              │
        └────────────┬────────────┘
                     │
              GovernanceProvider
              (generic interface)
                     │
            ┌────────┴────────┐
            │                 │
     SolventAdapter    Future adapters
            │
            ▼
                 SOLVENT
        ┌─────────────────────────┐
        │ Frozen authority kernel  │
        │ Evidence / belief / debt │
        │ Authority                │
        │ Intent                   │
        │ Authorization            │
        │ Execution                │
        └─────────────────────────┘
```

---

## 1. Repository Architecture

```text
conductor/
├── cmd/
│   └── conductor/
│       └── main.go                    # Binary entry point
├── internal/
│   ├── domain/
│   │   ├── project.go                 # Project entity
│   │   ├── task.go                    # Task entity
│   │   ├── activity.go                # Activity entity
│   │   └── dependency.go              # Task dependency entity
│   ├── governance/
│   │   ├── provider.go                # GovernanceProvider interface
│   │   ├── types.go                   # Generic governance types
│   │   └── null_provider.go           # No-op implementation
│   ├── adapter/
│   │   └── solvent/
│   │       ├── adapter.go             # Solvent GovernanceProvider impl
│   │       ├── client.go              # HTTP client to Solvent
│   │       └── translator.go          # Solvent concepts → generic types
│   ├── store/
│   │   ├── sqlite.go                  # SQLite connection and migrations
│   │   ├── project_repo.go            # Project CRUD
│   │   ├── task_repo.go               # Task CRUD + atomic claiming
│   │   ├── activity_repo.go           # Activity append-only log
│   │   └── dependency_repo.go         # Dependency CRUD
│   ├── service/
│   │   ├── project_service.go         # Project lifecycle orchestration
│   │   ├── task_service.go            # Task lifecycle orchestration
│   │   └── governance_service.go      # Governance projection reads
│   ├── api/
│   │   ├── server.go                  # HTTP server setup
│   │   ├── routes.go                  # Route definitions
│   │   ├── auth.go                    # Authentication middleware
│   │   ├── project_handler.go         # Project endpoints
│   │   ├── task_handler.go            # Task endpoints
│   │   └── governance_handler.go      # Governance endpoints
│   ├── mcp/
│   │   ├── server.go                  # MCP stdio server
│   │   └── tools.go                   # Tool definitions and handlers
│   └── web/
│       ├── server.go                  # Static file server
│       └── templates/
│           ├── layout.html            # Base template
│           ├── project.html           # Project dashboard
│           └── task.html              # Task detail
├── migrations/
│   ├── 001_initial.sql                # Core tables
│   └── 002_dependencies.sql           # Dependency table
├── go.mod
├── go.sum
├── Taskfile.yml                       # Build/test commands
└── plan2.md                           # This document
```

### Responsibility Boundaries

| Boundary | Responsibility | Owner |
|----------|----------------|-------|
| **Conductor** | Organizes and coordinates work | Conductor |
| **Agent** | Performs work and interacts with external systems | External |
| **Governance Provider** | Evaluates externally governed consequences | Interface in Conductor |
| **Solvent** | Authoritative implementation of governance provider | External |
| **Domain Application** | Supplies domain semantics and policy | External |

**Critical:** These are responsibility boundaries within ONE service, not five separate services.

---

## 2. Domain Model

### Project

| Property | Value |
|----------|-------|
| **Why it exists** | Groups related work items and provides context for agents |
| **What it owns** | Project identity, name, description, status, creation time |
| **What it must not own** | Authority, governance state, execution decisions, domain semantics |
| **Why in Conductor** | Project coordination is not a governance concern |
| **Type** | Durable state |

### Task

| Property | Value |
|----------|-------|
| **Why it exists** | Represents a unit of work that agents can claim and execute |
| **What it owns** | Task identity, title, description, status, priority, assignment, generic governance reference |
| **What it must not own** | Authority, belief state, execution authorization, domain-specific concepts |
| **Why in Conductor** | Task lifecycle management is project coordination, not governance |
| **Type** | Durable state |

### Activity

| Property | Value |
|----------|-------|
| **Why it exists** | Records what happened during task execution for audit and human visibility |
| **What it owns** | Immutable event log of task mutations and agent actions |
| **What it must not own** | Authority decisions, governance state changes, artifact storage |
| **Why in Conductor** | Activity logging for project events is distinct from governance audit |
| **Type** | Durable state (append-only) |

### TaskDependency

| Property | Value |
|----------|-------|
| **Why it exists** | Records blocking relationships between tasks |
| **What it owns** | Dependency relationships (task A blocks task B) |
| **What it must not own** | Dependency type semantics, domain-specific rules |
| **Why in Conductor** | Dependency tracking is project coordination |
| **Type** | Durable state |

### Agent (Not a Table)

| Property | Value |
|----------|-------|
| **Why it exists** | Represents an agent performing work |
| **What it owns** | Agent identity within Conductor (derived from authentication) |
| **What it must not own** | Solvent principal identity, authority, governance |
| **Why in Conductor** | Agent assignment and attribution are project concerns |
| **Type** | String identifier on Task (e.g., "agent-bob-1") |

### Entities NOT Included in v0

| Concept | Reason |
|---------|--------|
| Specification | Embedded in Task.description (Markdown) |
| Artifact | Recorded in Activity metadata; Conductor does not own artifact storage |
| Workflow | Future extension, not core coordination |
| Domain-specific concepts | Belongs in domain applications, not Conductor core |

---

## 3. SQLite Design

### Table: conductor_project

```sql
CREATE TABLE conductor_project (
    id          TEXT PRIMARY KEY,           -- UUID
    name        TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'completed', 'archived')),
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### Table: conductor_task

```sql
CREATE TABLE conductor_task (
    id              TEXT PRIMARY KEY,       -- UUID
    project_id      TEXT NOT NULL REFERENCES conductor_project(id),
    title           TEXT NOT NULL,
    description     TEXT,                   -- Markdown specification
    status          TEXT NOT NULL DEFAULT 'proposed'
                    CHECK (status IN ('proposed', 'active', 'review', 'accepted', 'rejected', 'blocked', 'cancelled')),
    priority        TEXT NOT NULL DEFAULT 'medium'
                    CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    current_agent   TEXT,                   -- Agent identifier (nullable)
    governance_ref  TEXT,                   -- Opaque JSON governance reference (nullable)
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_task_project ON conductor_task(project_id);
CREATE INDEX idx_task_status ON conductor_task(status);
CREATE INDEX idx_task_agent ON conductor_task(current_agent);
```

**governance_ref is opaque JSON:**

```json
{
    "provider": "solvent",
    "reference_id": "scenario-abc-123",
    "metadata": {}
}
```

Conductor treats this as an opaque reference. The Solvent adapter interprets it. Conductor never reads the `reference_id` directly.

### Table: conductor_activity

```sql
CREATE TABLE conductor_activity (
    id          TEXT PRIMARY KEY,           -- UUID
    task_id     TEXT NOT NULL REFERENCES conductor_task(id),
    actor_type  TEXT NOT NULL CHECK (actor_type IN ('human', 'agent', 'system')),
    actor_id    TEXT NOT NULL,              -- Authenticated actor identifier
    action      TEXT NOT NULL,              -- Event type (see lifecycle)
    details     TEXT,                       -- JSON metadata (non-authoritative)
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_activity_task ON conductor_activity(task_id);
CREATE INDEX idx_activity_created ON conductor_activity(task_id, created_at DESC);
```

**Activity semantics:**
- `event`: Something happened (task claimed, status changed)
- `reference`: External reference (commit hash, PR URL) — not owned by Conductor
- `metadata`: Non-authoritative context — never used for authorization

Conductor does NOT own artifact storage. Activity records events, not durable artifacts.

### Table: conductor_dependency

```sql
CREATE TABLE conductor_dependency (
    id              TEXT PRIMARY KEY,       -- UUID
    task_id         TEXT NOT NULL REFERENCES conductor_task(id),
    blocked_by_id   TEXT NOT NULL REFERENCES conductor_task(id),
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(task_id, blocked_by_id)
);

CREATE INDEX idx_dependency_task ON conductor_dependency(task_id);
CREATE INDEX idx_dependency_blocked_by ON conductor_dependency(blocked_by_id);
```

### Invariants

1. **Single source of truth:** Conductor owns project/task/activity/dependency state. Governance provider owns governance state.
2. **No duplicate truth:** Conductor never stores belief status, debt state, authority, or authorization.
3. **Append-only activity:** Activity records are never updated or deleted.
4. **Referential integrity:** Task must belong to a Project. Activity must belong to a Task. Dependencies reference valid Tasks.
5. **Governance projection is computed:** Never stored. Always derived from governance provider queries.
6. **governance_ref is opaque:** Conductor never interprets its contents.
7. **Atomic task claiming:** Only one agent can claim a proposed task (see lifecycle).

---

## 4. Project and Task Lifecycle

### Task Status Machine

```text
proposed
    ↓ (agent claims)
active
    ↓ (agent submits for review)
review
    ├── accepted (terminal)
    └── rejected → active

proposed → blocked (external dependency)
active → blocked (external dependency)
blocked → active (dependency resolved)

proposed → cancelled (human/agent cancels)
active → cancelled (human/agent cancels)
review → cancelled (human cancels)
blocked → cancelled (human cancels)
```

### Status Definitions

| Status | Meaning | Terminal |
|--------|---------|----------|
| **proposed** | Work defined, not yet claimed | No |
| **active** | Agent is working on the task | No |
| **review** | Work submitted, awaiting judgment | No |
| **accepted** | Work verified and accepted | Yes |
| **rejected** | Work needs rework, returns to active | No |
| **blocked** | Cannot proceed due to external dependency | No |
| **cancelled** | Work abandoned | Yes |

### Allowed Transitions

| From | To | Trigger | Actor | Activity Event |
|------|----|---------|-------|----------------|
| proposed | active | Agent claims task | agent | task.claimed |
| proposed | blocked | External dependency identified | agent, system | task.blocked |
| proposed | cancelled | Human/agent cancels | human, agent | task.cancelled |
| active | review | Agent submits for review | agent | task.submitted |
| active | blocked | External dependency identified | agent, system | task.blocked |
| active | cancelled | Human/agent cancels | human, agent | task.cancelled |
| review | accepted | Reviewer approves | human, agent | task.accepted |
| review | rejected | Reviewer rejects | human, agent | task.rejected |
| review | cancelled | Human cancels | human | task.cancelled |
| blocked | active | Dependency resolved | agent, human | task.unblocked |
| blocked | cancelled | Human cancels | human | task.cancelled |

### Invalid Transitions

- proposed → review (must be active first)
- proposed → accepted (must be active first)
- proposed → rejected (must be active first)
- active → accepted (must go through review)
- review → active (must be rejected, not directly return)
- blocked → review (must be active first)
- Any terminal state → any non-terminal state

### Atomic Task Claiming

Task claiming uses SQLite compare-and-swap to prevent races:

```sql
UPDATE conductor_task
SET current_agent = ?,
    status = 'active',
    updated_at = datetime('now')
WHERE id = ?
  AND status = 'proposed'
  AND current_agent IS NULL;
```

Verify exactly one row changed. If zero rows changed, the claim failed (another agent claimed it or task is not in proposed state).

---

## 5. Agent Model

### Agent Identity (Project Layer)

```text
agent_id    TEXT    -- Derived from authentication, not caller-supplied
agent_type  TEXT    -- "coding_agent", "human" (for v0)
```

**Critical:** Agent identity is derived from the authenticated actor at the Conductor boundary, NOT from arbitrary request metadata.

### Authentication Flow

```text
Agent sends request with API key or MCP trusted process
    ↓
Conductor authenticates at boundary
    ↓
Conductor derives agent_id from authenticated identity
    ↓
Agent identity is used for all operations
```

**NOT:**

```text
Request says agent_id = "bob"
    ↓
Conductor trusts it
```

### Agent Assignment

- Agent claims a task via API/MCP: `conductor_claim_task(task_id)`
- Agent identity is derived from authentication
- Task.current_agent is set to authenticated agent_id
- Activity event `task.claimed` is recorded
- Atomic claiming prevents two agents from claiming simultaneously

### Agent Status (Derived)

- **idle:** No task assigned (no tasks with current_agent = agent_id AND status = 'active')
- **working:** Task assigned (tasks with current_agent = agent_id AND status = 'active')
- **blocked:** Task blocked (tasks with current_agent = agent_id AND status = 'blocked')

Derive from Task state, not stored separately.

### Agent-Produced Artifacts

Recorded in Activity as metadata (references, not owned artifacts):

```json
{
    "action": "artifact_reference",
    "details": {
        "type": "code_commit",
        "url": "https://github.com/...",
        "hash": "abc123",
        "note": "Conductor records reference, not artifact ownership"
    }
}
```

### Separation from Solvent Principal

- **Project-layer agent identity:** For attribution and assignment within Conductor
- **Solvent principal identity:** For authority and governance within Solvent
- When agent performs consequential action through Solvent, Solvent principal is created separately at governance boundary
- The two identities are not silently equated

---

## 6. Agent API and MCP

### Minimal API Surface

| Operation | Method | Path | Purpose |
|-----------|--------|------|---------|
| get_project | GET | /v1/projects/:id | Read project context |
| list_tasks | GET | /v1/projects/:id/tasks | List tasks in project |
| create_task | POST | /v1/projects/:id/tasks | Create new task |
| get_task | GET | /v1/tasks/:id | Read task detail |
| update_task | PATCH | /v1/tasks/:id | Update task status/progress |
| claim_task | POST | /v1/tasks/:id/claim | Agent claims task (atomic) |
| submit_task | POST | /v1/tasks/:id/submit | Agent submits for review |
| accept_task | POST | /v1/tasks/:id/accept | Reviewer accepts work |
| reject_task | POST | /v1/tasks/:id/reject | Reviewer rejects work |
| report_blocker | POST | /v1/tasks/:id/blocker | Report blocking issue |
| resolve_blocker | POST | /v1/tasks/:id/resolve | Resolve blocking issue |
| post_activity | POST | /v1/tasks/:id/activity | Record activity |
| get_governance | GET | /v1/tasks/:id/governance | Get governance status |
| next_task | GET | /v1/projects/:id/next | Get next unblocked task |

### MCP Tools

```text
conductor_get_project       -- Read project context
conductor_list_tasks        -- List tasks in project
conductor_get_task          -- Read specific task
conductor_create_task       -- Create new task
conductor_update_task       -- Update task status/progress
conductor_claim_task        -- Agent claims task (atomic)
conductor_submit_task       -- Agent submits for review
conductor_accept_task       -- Reviewer accepts work
conductor_reject_task       -- Reviewer rejects work
conductor_report_blocker    -- Report blocking issue
conductor_resolve_blocker   -- Resolve blocking issue
conductor_post_activity     -- Record activity
conductor_get_governance    -- Get governance status
conductor_next_task         -- Get next unblocked task
```

### Operation Details

#### claim_task

- **Purpose:** Agent claims ownership of a task
- **Actor:** Authenticated agent (derived from authentication)
- **Input:** task_id
- **Output:** Updated task
- **State mutations:** Task.current_agent = agent_id, Task.status = 'active' (if proposed)
- **Activity event:** task.claimed
- **Governance interaction:** None
- **Concurrency:** Atomic compare-and-swap prevents races

#### submit_task

- **Purpose:** Agent submits work for review
- **Actor:** Assigned agent only
- **Input:** task_id
- **Output:** Updated task
- **State mutations:** Task.status = 'review'
- **Activity event:** task.submitted
- **Governance interaction:** None

#### accept_task

- **Purpose:** Reviewer accepts completed work
- **Actor:** Human or reviewer agent
- **Input:** task_id
- **Output:** Updated task
- **State mutations:** Task.status = 'accepted'
- **Activity event:** task.accepted
- **Governance interaction:** None

#### get_governance

- **Purpose:** Read governance status for a task
- **Actor:** Agent, Human
- **Input:** task_id
- **Output:** Governance status projection
- **State mutations:** None
- **Activity event:** None
- **Governance interaction:** Read-only query to governance provider

---

## 7. Governance Provider Interface

### Generic Interface

```go
// GovernanceProvider is the generic interface for external governance systems.
type GovernanceProvider interface {
    // GetState returns the current governance state for a reference.
    GetState(ctx context.Context, ref GovernanceReference) (*GovernanceState, error)
    
    // IsAuthorized checks if a specific action is permitted.
    IsAuthorized(ctx context.Context, ref GovernanceReference, action string) (*AuthorizationResult, error)
}

// GovernanceReference is an opaque reference to external governance context.
type GovernanceReference struct {
    Provider    string                 `json:"provider"`     // "solvent", "manual", etc.
    ReferenceID string                 `json:"reference_id"` // Opaque ID within provider
    Metadata    map[string]interface{} `json:"metadata"`     // Provider-specific, Conductor ignores
}

// GovernanceState represents the current state of external governance.
type GovernanceState struct {
    Reference   GovernanceReference `json:"reference"`
    Status      string              `json:"status"`      // "unknown", "pending", "ready", "blocked"
    Blockers    []string            `json:"blockers"`    // Generic reasons
    RefreshedAt time.Time           `json:"refreshed_at"`
}

// AuthorizationResult represents whether an action is permitted.
type AuthorizationResult struct {
    Allowed   bool     `json:"allowed"`
    Reason    string   `json:"reason"`    // Human-readable, generic
    Providers []string `json:"providers"` // Which systems were consulted
}
```

### Null Provider (Default)

```go
// NullProvider is the default governance provider when no external governance is configured.
type NullProvider struct{}

func (p *NullProvider) GetState(ctx context.Context, ref GovernanceReference) (*GovernanceState, error) {
    return &GovernanceState{
        Reference:   ref,
        Status:      "unknown",
        Blockers:    []string{"no governance provider configured"},
        RefreshedAt: time.Now(),
    }, nil
}

func (p *NullProvider) IsAuthorized(ctx context.Context, ref GovernanceReference, action string) (*AuthorizationResult, error) {
    return &AuthorizationResult{
        Allowed:   false,
        Reason:    "no governance provider configured",
        Providers: []string{},
    }, nil
}
```

### Solvent Adapter (One Implementation)

```go
// SolventAdapter implements GovernanceProvider for Solvent.
type SolventAdapter struct {
    client *SolventClient
}

func (a *SolventAdapter) GetState(ctx context.Context, ref GovernanceReference) (*GovernanceState, error) {
    // 1. Call Solvent API with ref.ReferenceID
    // 2. Translate Solvent response to generic GovernanceState
    // 3. Return generic state
}

func (a *SolventAdapter) IsAuthorized(ctx context.Context, ref GovernanceReference, action string) (*AuthorizationResult, error) {
    // 1. Call Solvent API with ref.ReferenceID and action
    // 2. Translate Solvent response to generic AuthorizationResult
    // 3. Return generic result
}
```

**Translation layer** (inside adapter):

```go
// translateSolventState translates Solvent concepts to generic governance concepts.
func translateSolventState(solventState *SolventLedgerSnapshot) *GovernanceState {
    status := "unknown"
    blockers := []string{}
    
    switch {
    case !solventState.AuditClean:
        status = "blocked"
        blockers = append(blockers, "audit issues")
    case len(solventState.LiveIntents) > 0:
        status = "pending"
    case solventState.CanPromote:
        status = "ready"
    default:
        status = "blocked"
        // Add debt items as blockers
        for _, belief := range solventState.Beliefs {
            for _, debt := range belief.RemainingDebt {
                blockers = append(blockers, debt)
            }
        }
    }
    
    return &GovernanceState{
        Reference: GovernanceReference{
            Provider:    "solvent",
            ReferenceID: solventState.ScenarioID,
        },
        Status:      status,
        Blockers:    blockers,
        RefreshedAt: time.Now(),
    }
}
```

### Key Principle

> Solvent-specific vocabulary (belief, debt, authority, intent) NEVER appears in Conductor's generic interface.

The Solvent adapter translates Solvent concepts into generic governance concepts. Conductor only understands:
- `status`: unknown, pending, ready, blocked
- `blockers`: generic string reasons
- `allowed`: boolean authorization result
- `reason`: human-readable explanation

---

## 8. Governance Projection

### What Conductor Reads

From governance provider (via generic interface):
- Governance state (unknown, pending, ready, blocked)
- Blockers (generic string reasons)
- Authorization status (allowed/not allowed with reason)

### Semantic States

| State | Meaning | Conductor Behavior |
|-------|---------|-------------------|
| **unknown** | Cannot determine governance state | Mark as stale, do not block operations |
| **pending** | Governance in progress | Task may be blocked, show current state |
| **ready** | Governance satisfied | Task can proceed to consequential action |
| **blocked** | Governance not satisfied | Task is blocked, show blockers |

### When It Refreshes

- On explicit request (agent queries governance)
- On task status transition (if task has governance_ref)
- Not automatic polling (wasteful)

### Staleness Semantics

```json
{
    "reference": {
        "provider": "solvent",
        "reference_id": "scenario-abc-123"
    },
    "status": "ready",
    "blockers": [],
    "refreshed_at": "2026-09-10T14:30:00Z",
    "stale": false
}
```

**Freshness rules:**
- If `refreshed_at` < 30 seconds ago: `stale: false`
- If `refreshed_at` >= 30 seconds ago: `stale: true`
- If provider unavailable: return last-known state with `stale: true`
- If no previous state: return `status: "unknown"`

### Critical Constraint

> A governance projection is never itself authorization.

Conductor displays governance state for situational awareness. It never uses governance state to permit or deny actions. Authorization is the governance provider's responsibility.

---

## 9. Consequential Action Boundary

### Generic Model

```text
Ordinary work
    ↓
Task reaches externally governed boundary
    ↓
Governance reference/provider becomes relevant
    ↓
External governance system determines what is permitted
```

**Conductor does NOT:**
- Classify what is "consequential"
- Decide which actions require governance
- Contain domain-specific policy

**Domain applications DO:**
- Mark tasks with governance references when appropriate
- Define which actions require governance
- Supply domain-specific policy

### How It Works

1. Domain application creates task with `governance_ref` when external governance applies
2. Conductor detects `governance_ref` is present
3. Conductor queries governance provider for status
4. Governance provider determines what is permitted
5. Agent/human acts based on governance provider response

### Conductor's Role

Conductor coordinates the transition between:
- Ordinary work (no governance reference)
- Governed work (governance reference present)

Conductor does NOT classify or judge what is consequential. That decision belongs to domain applications.

---

## 10. Minimal UI

### Single Page: Project Dashboard

```text
┌─────────────────────────────────────────────────────────────────┐
│ PROJECT: Feature Development                                    │
│ Status: Active | Tasks: 12 | In Progress: 3 | Blocked: 1        │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│ TASK BOARD                                                        │
│                                                                   │
│ [Proposed]        [Active]         [Review]       [Accepted]     │
│                                                                [x]│
│ ┌──────────┐    ┌──────────┐     ┌──────────┐   ┌──────────┐   │
│ │ Task A   │    │ Task B   │     │ Task C   │   │ Task D   │   │
│ │ agent: * │    │ agent: * │     │ pending  │   │ accepted │   │
│ │ 2h ago   │    │ now      │     │ review   │   │          │   │
│ └──────────┘    └──────────┘     └──────────┘   └──────────┘   │
│                                                                   │
│ GOVERNANCE PANEL                                                  │
│                                                                   │
│ Provider: solvent                                                 │
│ Status: ready                                                     │
│ Blockers: none                                                    │
│ Last checked: 2 minutes ago                                       │
│                                                                   │
│ ACTIVITY LOG (recent)                                             │
│                                                                   │
│ [14:23] Agent bob-1 completed Task D                             │
│ [14:21] Agent bob-1 started Task B                               │
│ [14:18] Human accepted Task C                                    │
│ [14:15] Agent bob-1 reported blocker on Task C                   │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Views

1. **Project Overview** - Project name, status, task counts
2. **Task Board** - Kanban-style view of tasks by status
3. **Governance Panel** - Governance status for project
4. **Activity Stream** - Recent events across all tasks

### Human Can Understand

- What is being worked on? → Task board shows active tasks
- Who/what is doing it? → current_agent shown on each task
- What happened? → Activity stream shows recent events
- What is blocked? → Blocked column shows blocked tasks
- What does governance currently allow? → Governance panel shows status
- Why can't the next step happen? → Blockers column shows reasons

### What UI Does NOT Require

- Clicking through every agent action
- Managing every evidence item
- Approving every workflow transition
- Manually updating task status

Human intervention happens when:
- A task is blocked by governance and needs human judgment
- An authorization boundary requires human approval
- The agent asks for clarification

---

## 11. End-to-End Workflow (Domain-Neutral)

### Generic SDLC Workflow

```text
1. Create project
   → Project exists with name and description

2. Decompose work
   → Create tasks with descriptions and dependencies

3. Agent claims task
   → Atomic claim, task moves to active

4. Agent works
   → Posts activity, references artifacts

5. Agent submits for review
   → Task moves to review

6. Reviewer accepts/rejects
   → Task moves to accepted or back to active

7. If governed consequence requested
   → Governance reference present on task
   → Query governance provider
   → Governance provider evaluates
   → Returns authorization status

8. If authorized
   → Execute external action via governance provider
   → Record outcome in activity

9. Task completed
   → Task moves to accepted
```

### What This Demonstrates

- Conductor coordinates work
- Agents perform engineering tasks
- Governance provider evaluates consequences
- Humans intervene only for genuine judgment (review, approval)
- Full audit trail of all actions
- Generic model works for any domain

---

## 12. Failure and Race Conditions

| Failure | Source of Truth | Conductor State | Governance State | Recovery | Human Visible |
|---------|-----------------|-----------------|------------------|----------|---------------|
| Agent crashes mid-task | Task status in Conductor DB | Task remains 'active' | No change | Task timeout or reassignment | Task shown as active but stale |
| Agent reports false completion | Activity log + human review | Task moves to 'review' | No change | Human reviews, rejects | Task rejected, returns to active |
| Agent abandons task | Task status in Conductor DB | Task remains 'active' | No change | Task timeout or reassignment | Task shown as active but stale |
| Two agents claim same task | Task.current_agent | Atomic claim ensures only one succeeds | No change | Second claim rejected | Second agent receives error |
| Conductor loses connection to governance | Conductor continues | Governance projection shows "unknown" | Governance provider continues | Conductor reconnects | Governance panel shows stale |
| Governance state changes while agent works | Governance provider is authoritative | Governance projection shows current | New state | Agent re-reads governance | Governance panel updates |
| Governance becomes stale | Governance provider timestamps | Governance projection shows stale | No change | Agent re-reads before consequential action | Governance panel shows stale |
| Governance unavailable | Governance provider | Governance projection shows "unknown" | No change | Agent receives degraded status | Governance panel shows unknown |
| Execution fails | Governance provider | Task remains in current state | Execution failed | Agent retries or reports blocker | Task blocked with error |
| Execution result ambiguous | Governance provider | Task remains in current state | Execution ambiguous | Agent re-checks status | Task shown as current state |
| Project DB unavailable | Solvent continues | Conductor returns 503 | Solvent continues | Conductor reconnects | HTTP 503 error |
| Agent is replaced | Agent identity in Conductor | Task.current_agent updated | No change | New agent claims task | Task shows new agent |

---

## 13. Testing Strategy

### Test Types

| Test Type | SQLite | Governance Provider | Agent |
|-----------|--------|---------------------|-------|
| Unit | In-memory | Stub | N/A |
| Repository | Real file | Stub | N/A |
| API | In-memory | Stub | N/A |
| MCP | In-memory | Stub | N/A |
| Adapter | In-memory | Real or stub | N/A |
| Integration | Real file | Real | Fake |
| E2E | Real file | Real | Fake |

### Test Categories

- **Unit tests:** `internal/domain/*_test.go` - Entity validation
- **Repository tests:** `internal/store/*_test.go` - CRUD operations against real SQLite
- **Service tests:** `internal/service/*_test.go` - Service orchestration with stubs
- **API tests:** `internal/api/*_test.go` - HTTP endpoints with httptest
- **MCP tests:** `internal/mcp/*_test.go` - Tool verification over stdio
- **Adapter tests:** `internal/adapter/solvent/*_test.go` - Solvent translation layer
- **Governance tests:** Governance projection logic
- **Integration tests:** Full workflow with real SQLite and stub governance
- **E2E tests:** Complete workflow from project creation to task completion

### Invariant Tests

1. All project writes through SQLite transactions
2. No project code imports Solvent kernel directly
3. Governance projection is always derived, never stored
4. No duplicate truth between project layer and governance provider
5. Activity records are append-only
6. Task claiming is atomic (compare-and-swap)
7. Agent identity is derived from authentication, not request metadata
8. governance_ref is never interpreted by Conductor core

---

## 14. Implementation Phases

### Phase 0: Repository Audit and Architecture Reconciliation

**Objective:** Establish clean separation between Conductor and Solvent.

**Scope:**
- Create repository structure
- Create go.mod
- Create basic Taskfile.yml
- Verify no Solvent imports

**Files/Packages Introduced:**
- `go.mod`
- `Taskfile.yml`
- `cmd/conductor/main.go` (stub)
- `internal/domain/` (entity definitions)
- `internal/governance/` (interface + types)

**Schema Changes:** None

**Interfaces Introduced:**
- `GovernanceProvider` interface
- `GovernanceReference` type
- `GovernanceState` type
- `AuthorizationResult` type

**Tests Required:**
- `go build ./...` succeeds
- `go vet ./...` clean
- No Solvent imports verified

**Acceptance Criteria:**
- Repository compiles
- No kernel imports
- GovernanceProvider interface defined

**Dependencies:** None

**Non-Goals:** No persistence, no API, no MCP

---

### Phase 1: Project Domain Model and Persistence

**Objective:** Project, Task, Activity, Dependency CRUD with SQLite persistence.

**Scope:**
- SQLite schema creation
- Repository pattern implementation
- CRUD operations for all entities
- Atomic task claiming

**Files/Packages Introduced:**
- `migrations/001_initial.sql`
- `migrations/002_dependencies.sql`
- `internal/store/sqlite.go`
- `internal/store/project_repo.go`
- `internal/store/task_repo.go`
- `internal/store/activity_repo.go`
- `internal/store/dependency_repo.go`

**Schema Changes:**
- conductor_project table
- conductor_task table
- conductor_activity table
- conductor_dependency table

**Interfaces Introduced:**
- `ProjectRepository`
- `TaskRepository`
- `ActivityRepository`
- `DependencyRepository`

**Tests Required:**
- All CRUD operations
- Lifecycle transitions
- Activity append-only invariant
- Atomic task claiming
- Dependency integrity

**Acceptance Criteria:**
- All tests pass
- Schema applies idempotently
- CRUD operations work correctly
- Atomic claiming prevents races

**Dependencies:** Phase 0

**Non-Goals:** No API, no MCP, no governance integration

---

### Phase 2: Agent API / MCP Interface

**Objective:** Agents can read and update project state.

**Scope:**
- HTTP REST API implementation
- MCP stdio server implementation
- Authentication middleware
- Actor identity derivation

**Files/Packages Introduced:**
- `internal/api/server.go`
- `internal/api/routes.go`
- `internal/api/auth.go`
- `internal/api/project_handler.go`
- `internal/api/task_handler.go`
- `internal/api/governance_handler.go`
- `internal/mcp/server.go`
- `internal/mcp/tools.go`
- `cmd/conductor/main.go` (full implementation)

**Schema Changes:** None

**Interfaces Introduced:**
- REST API endpoints
- MCP tools
- Authentication middleware

**Tests Required:**
- All API endpoints
- MCP tool verification
- Authentication
- Actor identity derivation

**Acceptance Criteria:**
- API endpoints work
- MCP tools verify
- Authentication works
- Actor identity is derived from authentication

**Dependencies:** Phase 1

**Non-Goals:** No governance integration, no UI

---

### Phase 3: Governance Provider Integration

**Objective:** Conductor can read governance state via generic interface.

**Scope:**
- NullProvider implementation
- SolventAdapter implementation
- Governance service layer
- Caching with freshness semantics

**Files/Packages Introduced:**
- `internal/governance/null_provider.go`
- `internal/adapter/solvent/adapter.go`
- `internal/adapter/solvent/client.go`
- `internal/adapter/solvent/translator.go`
- `internal/service/governance_service.go`

**Schema Changes:** None

**Interfaces Introduced:**
- `NullProvider` (default no-op)
- `SolventAdapter` (Solvent implementation)

**Tests Required:**
- NullProvider behavior
- SolventAdapter translation
- Governance service
- Freshness/staleness semantics
- Provider unavailability handling

**Acceptance Criteria:**
- NullProvider returns unknown state
- SolventAdapter translates correctly
- Freshness semantics work
- Provider unavailability handled gracefully

**Dependencies:** Phase 2

**Non-Goals:** No UI

---

### Phase 4: Minimal UI

**Objective:** Human can see project state and intervene when needed.

**Scope:**
- Server-rendered HTML templates
- Static file server
- Project dashboard page

**Files/Packages Introduced:**
- `internal/web/server.go`
- `internal/web/templates/layout.html`
- `internal/web/templates/project.html`
- `internal/web/templates/task.html`

**Schema Changes:** None

**Interfaces Introduced:** None

**Tests Required:** Manual acceptance test

**Acceptance Criteria:**
- Human can open browser
- See project state
- See agent activity
- See governance status

**Dependencies:** Phase 3

**Non-Goals:** No complex frontend framework, no real-time updates (use polling)

---

### Phase 5: Agent Integration Test with Stub

**Objective:** End-to-end test without a real coding agent.

**Scope:**
- Agent stub implementation
- Stub follows MCP protocol
- Stub completes synthetic task

**Files/Packages Introduced:**
- `internal/agentstub/stub.go`

**Schema Changes:** None

**Interfaces Introduced:** None

**Tests Required:**
- Stub completes task lifecycle
- Governance interaction works
- Review flow works

**Acceptance Criteria:** Stub completes synthetic task end-to-end

**Dependencies:** Phase 4

**Non-Goals:** No real agent integration

---

### Phase 6: End-to-End Synthetic SDLC Workflow

**Objective:** Full workflow from project creation to task completion.

**Scope:**
- Complete workflow demonstration
- Governance intersection
- Human review

**Files/Packages Introduced:** None (uses existing code)

**Schema Changes:** None

**Interfaces Introduced:** None

**Tests Required:** Complete workflow test

**Acceptance Criteria:**
- Complete workflow runs without errors
- All state transitions observable
- Governance intersection demonstrated
- Review flow works correctly

**Dependencies:** Phase 5

**Non-Goals:** No theatrical demo, no manual predetermination

---

### Phase 7: Domain Extension Readiness

**Objective:** Verify scaffolding supports arbitrary domain-specific applications.

**Scope:**
- Document generic extension points
- Verify all core entities are domain-neutral
- Identify gaps

**Files/Packages Introduced:** None

**Schema Changes:** None

**Interfaces Introduced:** None

**Tests Required:**
- Domain extension readiness checklist
- Verify domain-agnosticity

**Acceptance Criteria:**
- Arbitrary domain application can use Conductor without changing core model
- No domain-specific tables needed
- No domain-specific task states needed
- No domain-specific API changes needed

**Dependencies:** Phase 6

**Non-Goals:** No domain-specific implementation

---

## 15. Domain Extension Readiness

### Test: Arbitrary Domain Application

Imagine Conductor is used to build a completely unrelated software system tomorrow.

**Verification checklist:**

| Requirement | Status |
|-------------|--------|
| No domain-specific tables needed | ✅ Core model is generic |
| No domain-specific task states needed | ✅ Lifecycle is generic |
| No domain-specific API changes needed | ✅ API is generic |
| No domain-specific MCP tools needed | ✅ Tools are generic |
| No domain-specific policy embedded | ✅ Policy belongs in domain application |
| No Conductor core modification needed | ✅ Extension via governance_ref |

### How Domain Applications Consume Conductor

1. **Create project** with domain-neutral name and description
2. **Create tasks** with domain-neutral titles and descriptions
3. **Mark governed tasks** with `governance_ref` when external governance applies
4. **Configure governance provider** that understands domain-specific semantics
5. **Domain application** supplies domain-specific policy and interpretation

### What Belongs in Domain Applications

- Domain-specific trust semantics
- Domain-specific verification logic
- Domain-specific evidence types
- Domain-specific authority models
- Domain-specific workflow rules
- Domain-specific UI extensions

### What Belongs in Conductor

- Project lifecycle management
- Task lifecycle management
- Agent coordination
- Activity logging
- Dependency tracking
- Generic governance integration
- Domain-neutral UI

---

## 16. Security and Trust Boundaries

### Trust Boundaries

| Boundary | Trust Model |
|----------|-------------|
| Human → Conductor | Authenticated via API key (deployment boundary) |
| Agent → Conductor | Authenticated via API key or MCP trusted local |
| Conductor → Governance Provider | Read-only queries, never writes |
| Governance Provider → Conductor | Returns state, never commands |
| Agent → Governance Provider | Via Conductor, never direct |
| Governance Provider → Solvent | Solvent-specific adapter |

### Authentication

- **Human:** API key in HTTP header
- **Agent:** API key or MCP trusted local process
- **Conductor → Governance Provider:** API key or local process

### Attribution

- All activities recorded with actor_type and actor_id
- Agent activities attributed to authenticated agent_id
- Human activities attributed to authenticated human identifier
- System activities attributed to "system"

### Authorization

- Conductor manages project/task lifecycle
- Governance provider manages authority and governance
- Conductor never decides "this is authorized"
- Governance provider never decides "this task is complete"

### Responsibility

- **Conductor:** Project coordination, task lifecycle, activity logging
- **Governance Provider:** Authority, governance, consequential execution evaluation
- **Agent:** Performing work, interacting with external systems
- **Human:** Genuine judgment, authorization approvals, review decisions
- **Domain Application:** Domain semantics, domain policy, domain-specific logic

### Trusted vs Untrusted Input

- **Trusted:** Conductor internal state transitions, authenticated actor identity
- **Untrusted:** Agent-provided metadata, activity details, governance_ref content
- **Validated:** Task status transitions, actor identity derivation

---

## 17. Explicit Non-Goals

The following must NOT be implemented in v1:

- Full Jira replacement
- Multi-tenancy
- Complex RBAC
- Billing
- Notifications
- Chat platform
- General workflow engine
- Plugin marketplace
- Agent swarm framework
- Solvent kernel changes
- Solvent schema changes
- Solvent database access
- Domain-specific logic in Conductor core
- Large frontend framework
- Real-time collaboration (WebSocket)
- Document management
- Time tracking
- Reporting/analytics dashboard
- Mobile UI
- Internationalization
- SSO/OIDC
- Audit log export
- Webhook system
- Event bus
- Message queue
- Caching layer (beyond governance freshness)
- Rate limiting
- Metrics/monitoring
- Logging aggregation
- Container orchestration
- Kubernetes deployment
- CI/CD pipeline
- Automated testing in CI
- Code coverage
- Linting
- Formatting
- Documentation generation
- API versioning
- Deprecation handling
- Backward compatibility
- Migration tooling
- Database backup
- Database restore
- High availability
- Disaster recovery
- Performance optimization
- Load testing
- Stress testing
- Security auditing
- Penetration testing

---

## 18. Final Architecture Decision Record

| Decision | Choice | Alternative Rejected | Consequence |
|----------|--------|---------------------|-------------|
| **Repository** | Separate from Solvent | Same repo | Clean separation, independent evolution |
| **Binary** | Separate `cmd/conductor` | Part of `cmd/solvent-api` | Independent lifecycle, simpler API server |
| **Database** | Standalone SQLite | Same CockroachDB | Clear ownership, no shared schema |
| **Core entities** | Project, Task, Activity, Dependency (4 tables) | 3 tables with JSON | Relational integrity, better querying |
| **Agent model** | Authenticated identity derivation | Caller-supplied agent_id | Prevents spoofing, clean identity |
| **API** | REST + MCP | MCP only | Human and agent access |
| **MCP** | 14 tools | More tools | Minimal viable surface |
| **Governance boundary** | Generic GovernanceProvider interface | Solvent-specific interface | Domain-agnostic, extensible |
| **Governance projection** | Computed, never stored | Cached in DB | No duplicate truth |
| **Governance freshness** | Semantic states (fresh/stale/unknown) | Hard-coded TTL | Architecture-level clarity |
| **Task lifecycle** | PROPOSED → ACTIVE → REVIEW → ACCEPTED/REJECTED | PROPOSED → ACTIVE → COMPLETED | Clear review semantics |
| **UI** | Server-rendered HTML | SPA | Minimal complexity |
| **Execution boundary** | Agent + Governance Provider | Conductor executes | Clear responsibility |
| **Testing strategy** | Real SQLite + stub governance | Real governance for all | Deterministic tests |
| **Implementation sequence** | 8 phases | Different order | Logical dependency chain |
| **Major non-goals** | No Jira, no workflow engine, no Solvent changes | Various | Architectural focus |

---

## What Must Remain Outside Conductor

The following responsibilities belong OUTSIDE the frozen Conductor core:

1. **Domain-specific logic** — trust semantics, verification logic, domain policy
2. **Authority decisions** — governed by governance provider, not Conductor
3. **Artifact storage** — Conductor records references, not artifacts
4. **Governance interpretation** — Conductor shows state, doesn't judge
5. **Consequential action classification** — domain applications decide
6. **Solvent-specific concepts** — belief, debt, authority, intent belong in adapter
7. **Identity equivalence** — project agent ≠ Solvent principal

These are ALL responsibilities that belong in governance providers, domain applications, or external systems. Conductor coordinates work. Governance providers evaluate consequences. Domain applications supply meaning.

---

## Adversarial Revision Summary

### Architecture Retained

- Separate repository ✅
- Separate binary ✅
- Standalone SQLite ✅
- No Solvent DB access ✅
- Solvent remains authority ✅
- Conductor owns project state ✅
- Minimal UI ✅
- MCP + API ✅
- Server-rendered UI ✅
- Read-only governance integration ✅

### Architecture Changed

- Solvent-specific fields in Task → Generic governance_ref ✅
- Oracle-specific architecture → Domain Extension Readiness ✅
- Domain-specific consequence rules → Generic governance boundary ✅
- Hard-coded governance vocabulary → Semantic freshness states ✅
- JSON dependency representation → Relational task_dependency table ✅
- Agent-supplied identity → Authenticated identity derivation ✅
- 5-second cache as architecture → Fresh/stale/unknown semantics ✅
- False-completion lifecycle → PROPOSED → ACTIVE → REVIEW → ACCEPTED/REJECTED ✅
- Solvent-coupled interface → Generic GovernanceProvider ✅
- Five services → One service with internal boundaries ✅

### Domain-Specific Leakage Removed

- Oracle-specific domain mapping ✅
- Solvent concepts in Task model ✅
- Domain-specific consequence examples ✅
- Solvent vocabulary in generic interface ✅
- Domain-specific workflow examples ✅

### New Invariants

1. `governance_ref` is opaque JSON, never interpreted by Conductor core
2. `GovernanceProvider` uses only generic concepts (status, blockers, allowed)
3. Task claiming is atomic (compare-and-swap)
4. Agent identity is derived from authentication, not request metadata
5. Governance projection is never itself authorization
6. Activity records events, not artifacts
7. One service with internal boundaries, not five services

### Open Architectural Questions

1. **Governance freshness TTL:** Is 30 seconds appropriate, or should it be configurable?
2. **Dependency cycles:** Should Conductor detect circular dependencies, or reject them at creation?
3. **Task priority semantics:** Should priority affect scheduling, or is it purely informational?
4. **Activity retention:** Should old activities be pruned, or retained indefinitely?
5. **Governance provider failure:** Should Conductor block operations or proceed with degraded state?

### Implementation Readiness Verdict

**GREEN — Ready for implementation.**

The architecture is now genuinely domain-agnostic, with clean governance abstraction, precise lifecycle semantics, and explicit responsibility boundaries. The plan can proceed to Phase 0.

---

*This document was produced by adversarial review of the original implementation plan. No files were modified. No implementation was produced. This is a design specification awaiting final approval before implementation begins.*
