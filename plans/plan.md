# Conductor — Implementation Plan

## Architectural Decisions Already Made

- **Conductor is a standalone project layer**
- **Separate repository and separate binary/process from Solvent**
- **Standalone SQLite database** (not Solvent's CockroachDB)
- **Never directly read or write Solvent's database**
- **Never become a second authority engine**
- **Solvent remains authoritative** for governance/authorization state
- **Conductor remains authoritative** for project/work state

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
              Solvent adapter
              READ-ONLY boundary
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
│   │   └── agent.go                   # Agent identity model
│   ├── store/
│   │   ├── sqlite.go                  # SQLite connection and migrations
│   │   ├── project_repo.go            # Project CRUD
│   │   ├── task_repo.go               # Task CRUD
│   │   └── activity_repo.go           # Activity append-only log
│   ├── service/
│   │   ├── project_service.go         # Project lifecycle orchestration
│   │   ├── task_service.go            # Task lifecycle orchestration
│   │   └── governance_service.go      # Governance projection reads
│   ├── adapter/
│   │   └── solvent/
│   │       ├── adapter.go             # Solvent read-only adapter
│   │       ├── client.go              # HTTP/MCP client to Solvent
│   │       └── governance.go          # Governance state projection
│   ├── api/
│   │   ├── server.go                  # HTTP server setup
│   │   ├── routes.go                  # Route definitions
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
│   └── 001_initial.sql                # Schema creation
├── go.mod
├── go.sum
├── Taskfile.yml                       # Build/test commands
└── plan.md                            # This document
```

**Responsibilities:**

| Directory | Purpose |
|-----------|---------|
| `cmd/conductor/` | Binary entry point, dependency injection |
| `internal/domain/` | Entity definitions, no persistence logic |
| `internal/store/` | SQLite persistence, repository pattern |
| `internal/service/` | Business logic, lifecycle orchestration |
| `internal/adapter/solvent/` | Read-only Solvent integration |
| `internal/api/` | HTTP REST API |
| `internal/mcp/` | MCP stdio server for agent access |
| `internal/web/` | Minimal server-rendered UI |
| `migrations/` | Database schema |

---

## 2. Domain Model

### Project

| Property | Value |
|----------|-------|
| **Why it exists** | Groups related work items and provides context for agents |
| **What it owns** | Project identity, name, description, status, creation time |
| **What it must not own** | Authority, governance state, execution decisions |
| **Why in Conductor** | Project coordination is not a Solvent concern |
| **Type** | Durable state |

### Task

| Property | Value |
|----------|-------|
| **Why it exists** | Represents a unit of work that agents can claim and execute |
| **What it owns** | Task identity, title, description, status, priority, assignment, references to Solvent governance |
| **What it must not own** | Authority, belief state, execution authorization |
| **Why in Conductor** | Task lifecycle management is project coordination, not governance |
| **Type** | Durable state |

### Activity

| Property | Value |
|----------|-------|
| **Why it exists** | Records what happened during task execution for audit and human visibility |
| **What it owns** | Immutable event log of task mutations and agent actions |
| **What it must not own** | Authority decisions, governance state changes |
| **Why in Conductor** | Activity logging for project events is distinct from Solvent's audit log |
| **Type** | Durable state (append-only) |

### Agent (Not a Table in v0)

| Property | Value |
|----------|-------|
| **Why it exists** | Represents an agent performing work |
| **What it owns** | Agent identity within Conductor (not Solvent principal) |
| **What it must not own** | Solvent principal identity, authority, governance |
| **Why in Conductor** | Agent assignment and attribution are project concerns |
| **Type** | String identifier on Task (e.g., "ibm-bob-1") |

### Entities NOT Included in v0

| Concept | Reason |
|---------|--------|
| Specification | Embedded in Task.description (Markdown) |
| Artifact | Recorded in Activity as metadata |
| Dependency | Use blocked_by JSON array on Task for v0 |
| Workflow | Future extension, not core coordination |
| Governance Projection | Computed, never stored |

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
    id            TEXT PRIMARY KEY,         -- UUID
    project_id    TEXT NOT NULL REFERENCES conductor_project(id),
    title         TEXT NOT NULL,
    description   TEXT,                     -- Markdown specification
    status        TEXT NOT NULL DEFAULT 'proposed'
                  CHECK (status IN ('proposed', 'active', 'blocked', 'completed', 'cancelled')),
    priority      TEXT NOT NULL DEFAULT 'medium'
                  CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    current_agent TEXT,                     -- Agent identifier (nullable)
    scenario_id   TEXT,                     -- Solvent scenario reference (nullable)
    intent_id     TEXT,                     -- Solvent action_intent reference (nullable)
    blocked_by    TEXT,                     -- JSON array of task IDs (nullable)
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_task_project ON conductor_task(project_id);
CREATE INDEX idx_task_status ON conductor_task(status);
CREATE INDEX idx_task_agent ON conductor_task(current_agent);
```

### Table: conductor_activity

```sql
CREATE TABLE conductor_activity (
    id          TEXT PRIMARY KEY,           -- UUID
    task_id     TEXT NOT NULL REFERENCES conductor_task(id),
    actor_type  TEXT NOT NULL CHECK (actor_type IN ('human', 'agent', 'system')),
    actor_id    TEXT,                       -- Agent name or human identifier
    action      TEXT NOT NULL,              -- Event type
    details     TEXT,                       -- JSON metadata
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_activity_task ON conductor_activity(task_id);
CREATE INDEX idx_activity_created ON conductor_activity(task_id, created_at DESC);
```

### Invariants

1. **Single source of truth:** Conductor owns project/task/activity state. Solvent owns governance state.
2. **No duplicate truth:** Conductor never stores belief status, debt state, authority, or authorization.
3. **Append-only activity:** Activity records are never updated or deleted.
4. **Referential integrity:** Task must belong to a Project. Activity must belong to a Task.
5. **Governance projection is computed:** Never stored. Always derived from Solvent queries.

---

## 4. Project and Task Lifecycle

### Task Status Machine

```text
proposed
    ↓ (agent claims)
active
    ↓ (agent reports blocker)
blocked
    ↓ (blocker resolved)
active
    ↓ (agent completes)
completed

proposed → cancelled (human/agent cancels)
active → cancelled (human/agent cancels)
blocked → cancelled (human cancels)
```

### Allowed Transitions

| From | To | Trigger | Actor |
|------|----|---------|-------|
| proposed | active | Agent claims task | agent |
| proposed | cancelled | Human/agent cancels | human, agent |
| active | blocked | Agent reports blocker | agent |
| active | completed | Agent completes work | agent |
| active | cancelled | Human/agent cancels | human, agent |
| blocked | active | Blocker resolved | agent, human |
| blocked | cancelled | Human cancels | human |

### Invalid Transitions

- proposed → completed (must be active first)
- proposed → blocked (must be active first)
- completed → any (terminal state)
- cancelled → any (terminal state)

### Activity Events Recorded

| Transition | Activity Action |
|------------|----------------|
| proposed → active | task.claimed |
| proposed → cancelled | task.cancelled |
| active → blocked | task.blocked |
| active → completed | task.completed |
| active → cancelled | task.cancelled |
| blocked → active | task.unblocked |
| blocked → cancelled | task.cancelled |
| Any update | task.updated |

---

## 5. Agent Model

### Agent Identity (Project Layer)

```text
agent_id    TEXT    -- Unique identifier (e.g., "ibm-bob-1")
agent_type  TEXT    -- "coding_agent", "human" (for v0)
```

**Design:** Agent is a string identifier stored on Task.current_agent. Not a Solvent principal.

### Agent Assignment

- Agent claims a task via API/MCP: `conductor_claim_task(task_id, agent_id)`
- Task.current_agent is set to agent_id
- Activity event `task.claimed` is recorded
- Only one agent can claim a task at a time

### Agent Status (Derived)

- **idle:** No task assigned
- **working:** Task assigned (task.status = 'active')
- **blocked:** Task blocked (task.status = 'blocked')

Derive from Task state, not stored separately.

### Agent-Produced Artifacts

Recorded in Activity as metadata:

```json
{
  "action": "artifact_produced",
  "details": {
    "type": "code_commit",
    "url": "https://github.com/...",
    "hash": "abc123"
  }
}
```

### Separation from Solvent Principal

- **Project-layer agent identity:** For attribution and assignment within Conductor
- **Solvent principal identity:** For authority and governance within Solvent
- When agent performs consequential action through Solvent, Solvent principal is created separately at deployment/API boundary
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
| claim_task | POST | /v1/tasks/:id/claim | Agent claims task |
| complete_task | POST | /v1/tasks/:id/complete | Mark task done |
| report_blocker | POST | /v1/tasks/:id/blocker | Report blocking issue |
| post_activity | POST | /v1/tasks/:id/activity | Record activity |
| get_governance | GET | /v1/tasks/:id/governance | Get Solvent governance status |
| next_task | GET | /v1/projects/:id/next | Get next unblocked task |

### MCP Tools

```text
conductor_get_project       -- Read project context
conductor_list_tasks        -- List tasks in project
conductor_get_task          -- Read specific task
conductor_create_task       -- Create new task
conductor_update_task       -- Update task status/progress
conductor_claim_task        -- Agent claims task
conductor_complete_task     -- Mark task done
conductor_report_blocker    -- Report blocking issue
conductor_post_activity     -- Record activity
conductor_get_governance    -- Get Solvent governance status
conductor_next_task         -- Get next unblocked task
```

### Operation Details

#### claim_task

- **Purpose:** Agent claims ownership of a task
- **Actor:** Agent
- **Input:** task_id, agent_id
- **Output:** Updated task
- **State mutations:** Task.current_agent = agent_id, Task.status = 'active' (if proposed)
- **Activity event:** task.claimed
- **Solvent interaction:** None

#### complete_task

- **Purpose:** Agent marks task as done
- **Actor:** Agent
- **Input:** task_id, agent_id, optional result metadata
- **Output:** Updated task
- **State mutations:** Task.status = 'completed'
- **Activity event:** task.completed
- **Solvent interaction:** None (unless task has scenario_id, then read governance status)

#### report_blocker

- **Purpose:** Agent indicates task is blocked
- **Actor:** Agent
- **Input:** task_id, agent_id, reason
- **Output:** Updated task
- **State mutations:** Task.status = 'blocked'
- **Activity event:** task.blocked
- **Solvent interaction:** None

#### get_governance

- **Purpose:** Read Solvent governance status for a task
- **Actor:** Agent, Human
- **Input:** task_id
- **Output:** Governance status projection
- **State mutations:** None
- **Activity event:** None
- **Solvent interaction:** Read-only query to Solvent

---

## 7. Solvent Adapter

### Interface

```go
// SolventReader is the read-only interface to Solvent.
type SolventReader interface {
    // GetGovernanceStatus returns the governance state for a scenario.
    GetGovernanceStatus(ctx context.Context, scenarioID string) (*GovernanceStatus, error)

    // ExplainBelief returns why a belief is or is not promotable.
    ExplainBelief(ctx context.Context, beliefID string) (*BeliefExplanation, error)
}
```

### GovernanceStatus

```go
type GovernanceStatus struct {
    ScenarioID     string          `json:"scenario_id"`
    Beliefs        []BeliefStatus  `json:"beliefs"`
    HasLiveIntents bool            `json:"has_live_intents"`
    AuditClean     bool            `json:"audit_clean"`
    CanPromote     bool            `json:"can_promote"`
    CanAuthorize   bool            `json:"can_authorize"`
    Blockers       []string        `json:"blockers"`
}

type BeliefStatus struct {
    BeliefID      string   `json:"belief_id"`
    Claim         string   `json:"claim"`
    Status        string   `json:"status"`       // entered, promoted, retracted
    RemainingDebt []string `json:"remaining_debt"`
    CanPromote    bool     `json:"can_promote"`
    CanAuthorize  bool     `json:"can_authorize"`
}
```

### Error Handling

| Scenario | Behavior |
|----------|----------|
| Solvent unavailable | Return degraded status with "unknown" governance |
| Solvent timeout | Return last-known status if cached, else degraded |
| Invalid scenario_id | Return empty governance (no beliefs found) |
| Solvent returns error | Log error, return degraded status |

### Caching Rules

- Governance status is cached for 5 seconds
- Cache is invalidated on explicit refresh or after TTL
- Stale data is marked with `stale: true` in response
- Agents can force refresh via `force_refresh: true` parameter

### Critical Constraint

> Conductor must not infer or recreate authorization decisions that belong to Solvent.

Conductor reads governance state and displays it. It never decides "this is authorized" or "this can execute." That determination belongs to Solvent's kernel.

---

## 8. Governance Projection

### What Conductor Reads

From Solvent (via adapter):
- Belief status (entered, promoted, retracted)
- Debt items (remaining obligations)
- Authority state (active, revoked, stale)
- Intent state (none, live, executing, executed, cancelled)
- Execution state (none, success, rejected, ambiguous)

### How Conductor Maps Information

```text
Solvent Belief Status → Conductor Projection
─────────────────────────────────────────────
entered             → POSTULATED
promoted            → PROMOTED
retracted           → RETRACTED

Solvent Debt → Conductor Projection
─────────────────────────────────────────────
non-empty array     → OPEN
empty array         → CLEAR

Solvent Authority → Conductor Projection
─────────────────────────────────────────────
no activation       → NONE
activation exists   → ACTIVE
revocation exists   → REVOKED

Solvent Intent → Conductor Projection
─────────────────────────────────────────────
no intent           → NONE
intent.state=live   → LIVE
intent.state=executing → EXECUTING
intent.state=executed  → EXECUTED
intent.state=cancelled → CANCELLED
```

### When It Refreshes

- On explicit request (agent queries governance)
- On task status transition (if task has scenario_id)
- Not automatic polling (wasteful)

### Whether It Caches

- Yes, 5-second TTL cache
- Marked as stale if Solvent unavailable

### How Stale Information Is Marked

```json
{
  "scenario_id": "abc123",
  "stale": true,
  "last_updated": "2026-09-10T14:30:00Z",
  "beliefs": [...]
}
```

### What Happens When Solvent Is Unavailable

- Return last-known cached status with stale=true
- Log warning
- Do not block agent operations (project state is independent)

### What Happens When Governance State Changes While Agent Works

- Agent must re-read governance before consequential action
- Conductor does not push governance updates
- Agent is responsible for checking current state

---

## 9. Consequential Action Boundary

### Where Project Workflow Crosses Into Consequential Execution

```text
ORDINARY PROJECT WORKFLOW (Conductor owns):
──────────────────────────────────────────
Idea → Task created (proposed)
    → Agent claims task (active)
    → Agent works (editing, testing, reading)
    → Agent reports progress
    → Agent completes task (completed)

    NONE of this requires Solvent governance.

CONSEQUENTIAL EXECUTION (Solvent governs):
──────────────────────────────────────────
Task involves consequence:
    → Deploy to production
    → Grant access
    → Execute destructive operation
    → Change security policy
    → Publish artifact

    THEN Solvent governance becomes relevant:
    → Read governance status
    → Check if authority exists
    → If not, agent must work through Solvent lifecycle:
        → Create belief with evidence
        → Retire debt items
        → Promote belief
        → Create authority target
        → Attach justification
        → Request authorization
        → Approve target
        → Create intent
        → Execute authorized consequence
    → Task completes after execution
```

### How Two Lifecycles Interact

```text
Conductor Task Lifecycle          Solvent Governance Lifecycle
───────────────────────          ────────────────────────────
proposed                         (no interaction)
    ↓
active                           (no interaction)
    ↓
    ↓ agent determines task is consequential
    ↓
    ↓ ─────────────────────────→  scenario_id set on task
    ↓                             belief created
    ↓                             debt items assigned
    ↓                             belief promoted
    ↓                             authority target created
    ↓                             authorization approved
    ↓                             intent created
    ↓                             execution claimed
    ↓                             execution completed
    ↓
completed ←────────────────────  outcome recorded in Solvent
```

### Key Principle

> Do not collapse Conductor's workflow state and Solvent's governance lifecycle into a single state machine.

- **Conductor manages:** proposed → active → blocked → completed
- **Solvent manages:** evidence → belief → debt → promotion → authority → intent → authorization → execution

The two systems are independently owned. Conductor references Solvent via scenario_id. Solvent does not know about Conductor tasks.

---

## 10. Minimal UI

### Single Page: Project Dashboard

```text
┌─────────────────────────────────────────────────────────────────┐
│ PROJECT: Oracle Trust-but-Verify                                │
│ Status: Active | Tasks: 12 | In Progress: 3 | Blocked: 1        │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│ TASK BOARD                                                        │
│                                                                   │
│ [Proposed]        [Active]           [Blocked]        [Done]      │
│                                                                [x]│
│ ┌──────────┐    ┌──────────┐       ┌──────────┐     ┌──────────┐│
│ │ Task A   │    │ Task B   │       │ Task C   │     │ Task D   ││
│ │ agent: * │    │ agent: * │       │ BLOCKED  │     │ complete ││
│ │ 2h ago   │    │ now      │       │ gov: 235 │     │          ││
│ └──────────┘    └──────────┘       └──────────┘     └──────────┘│
│                                                                   │
│ GOVERNANCE PANEL                                                  │
│                                                                   │
│ Scenario: track1                                                 │
│ Beliefs: 2 promoted, 0 retracted                                 │
│ Live Intents: 1                                                  │
│ Audit: clean                                                     │
│ Blockers: none                                                   │
│                                                                   │
│ ACTIVITY LOG (recent)                                             │
│                                                                   │
│ [14:23] Agent bob-1 completed Task D                             │
│ [14:21] Agent bob-1 started Task B                               │
│ [14:18] Human promoted belief abc123                             │
│ [14:15] Agent bob-1 reported blocker on Task C                   │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Views

1. **Project Overview** - Project name, status, task counts
2. **Task Board** - Kanban-style view of tasks by status
3. **Governance Panel** - Solvent governance status for project
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

## 11. End-to-End SDLC Demonstration

### Workflow: Simple Feature Implementation with Governance

**Setup:**
1. Human creates project "Feature X" via Conductor API
2. Human creates task "Implement authentication module" (proposed)
3. Human creates task "Deploy to staging" (proposed, blocked by first task)

**Agent Work:**
4. Agent (Bob) claims task "Implement authentication module"
5. Agent reads task description, starts working
6. Agent writes code, runs tests, commits
7. Agent posts activity: "Code written, tests passing"
8. Agent completes task

**Governance Intersection:**
9. Agent claims task "Deploy to staging"
10. Agent determines this is consequential (deployment)
11. Agent reads governance status via Conductor
12. Agent sees no authority exists for deployment
13. Agent works through Solvent lifecycle:
    - Creates belief: "Code is ready for staging deployment"
    - Attaches evidence: test results, code review
    - Retires debt items
    - Promotes belief
    - Creates authority target
    - Attaches justification
    - Requests authorization
14. Human reviews and approves authority target
15. Agent creates intent
16. Agent executes deployment via Solvent executor
17. Agent posts activity: "Deployed to staging"
18. Agent completes task

**Human Review:**
19. Human opens Conductor UI
20. Human sees both tasks completed
21. Human sees governance panel shows authority was used
22. Human reviews activity log showing full audit trail

### What This Demonstrates

- Conductor coordinates work
- Agents perform engineering tasks
- Solvent governs consequential actions
- Humans intervene only for genuine judgment (approval)
- Full audit trail of all actions
- Two systems working together without collapsing responsibilities

---

## 12. Failure and Race Conditions

| Failure | Source of Truth | Conductor State | Solvent State | Recovery | Human Visible |
|---------|-----------------|-----------------|---------------|----------|---------------|
| Agent crashes mid-task | Task status in Conductor DB | Task remains 'active' | No change | Task timeout or reassignment | Task shown as active but stale |
| Agent reports false completion | Activity log + human review | Task marked 'completed' | No change | Human reviews, reopens | Task appears completed |
| Agent abandons task | Task status in Conductor DB | Task remains 'active' | No change | Task timeout or reassignment | Task shown as active but stale |
| Two agents claim same task | Task.current_agent | Only one agent can claim | No change | Second claim rejected 409 | Second agent receives error |
| Conductor loses connection to Solvent | Conductor continues | Governance projection shows "unknown" | Solvent continues | Conductor reconnects | Governance panel shows stale |
| Solvent state changes while agent works | Solvent is authoritative | Governance projection shows current | New state | Agent re-reads governance | Governance panel updates |
| Authority becomes stale | Solvent target_revocation | Governance projection shows revoked | Authority revoked | Agent re-reads before execution | Governance panel shows "REVOKED" |
| Authority is revoked | Solvent target_revocation | Governance projection shows revoked | Authority revoked | Agent receives denial | Task blocked, human notified |
| Wrong target requested | Solvent exact authority binding | No change | Solvent refuses | Agent receives error, reports blocker | Task blocked with error |
| Execution fails | Solvent execution outcome | Task remains 'active' or 'blocked' | Execution = rejected | Agent retries or reports blocker | Task blocked with error |
| Execution result ambiguous | Solvent execution outcome | Task remains 'active' | Execution = ambiguous | Agent re-checks status | Task shown as active |
| Project DB unavailable | Solvent continues | Conductor returns 503 | Solvent continues | Conductor reconnects | HTTP 503 error |
| Agent is replaced | Agent identity in Conductor | Task.current_agent updated | No change | New agent claims task | Task shows new agent |

---

## 13. Testing Strategy

### Test Types

| Test Type | SQLite | Solvent | Agent |
|-----------|--------|---------|-------|
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
- **Adapter tests:** `internal/adapter/solvent/*_test.go` - Read-only adapter behavior
- **Governance tests:** Governance projection logic
- **Integration tests:** Full workflow with real SQLite and stub Solvent
- **E2E tests:** Complete workflow from project creation to task completion

### Invariant Tests

1. All project writes through SQLite transactions
2. No project code imports Solvent kernel directly
3. Governance projection is always derived, never stored
4. No duplicate truth between project layer and Solvent
5. Activity records are append-only

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

**Schema Changes:** None

**Interfaces Introduced:** None

**Tests Required:**
- `go build ./...` succeeds
- `go vet ./...` clean
- No Solvent imports verified

**Acceptance Criteria:**
- Repository compiles
- No kernel imports
- Architecture documented

**Dependencies:** None

**Non-Goals:** No persistence, no API, no MCP

---

### Phase 1: Project Domain Model and Persistence

**Objective:** Project, Task, Activity CRUD with SQLite persistence.

**Scope:**
- SQLite schema creation
- Repository pattern implementation
- CRUD operations for all entities

**Files/Packages Introduced:**
- `migrations/001_initial.sql`
- `internal/store/sqlite.go`
- `internal/store/project_repo.go`
- `internal/store/task_repo.go`
- `internal/store/activity_repo.go`

**Schema Changes:**
- conductor_project table
- conductor_task table
- conductor_activity table

**Interfaces Introduced:**
- `ProjectRepository`
- `TaskRepository`
- `ActivityRepository`

**Tests Required:**
- All CRUD operations
- Lifecycle transitions
- Activity append-only invariant

**Acceptance Criteria:**
- All tests pass
- Schema applies idempotently
- CRUD operations work correctly

**Dependencies:** Phase 0

**Non-Goals:** No API, no MCP, no Solvent integration

---

### Phase 2: Agent API / MCP Interface

**Objective:** Agents can read and update project state.

**Scope:**
- HTTP REST API implementation
- MCP stdio server implementation
- Authentication middleware

**Files/Packages Introduced:**
- `internal/api/server.go`
- `internal/api/routes.go`
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

**Tests Required:**
- All API endpoints
- MCP tool verification
- Authentication

**Acceptance Criteria:**
- API endpoints work
- MCP tools verify
- Authentication works

**Dependencies:** Phase 1

**Non-Goals:** No Solvent integration, no governance projection, no UI

---

### Phase 3: Solvent Integration Adapter

**Objective:** Conductor can read Solvent governance state.

**Scope:**
- Solvent read-only adapter
- HTTP/MCP client to Solvent
- Error handling and caching

**Files/Packages Introduced:**
- `internal/adapter/solvent/adapter.go`
- `internal/adapter/solvent/client.go`
- `internal/adapter/solvent/governance.go`

**Schema Changes:** None

**Interfaces Introduced:**
- `SolventReader` interface
- `GovernanceStatus` struct

**Tests Required:**
- Adapter read-only behavior
- Error handling
- Caching
- Stale data handling

**Acceptance Criteria:**
- Governance projection works
- No Solvent writes observed
- Error handling works

**Dependencies:** Phase 2

**Non-Goals:** No governance projection UI, no UI

---

### Phase 4: Governance Projection

**Objective:** Compact governance view for project tasks.

**Scope:**
- Governance status projection logic
- Task status to governance mapping
- Next actionable task logic

**Files/Packages Introduced:**
- `internal/service/governance_service.go`

**Schema Changes:** None

**Interfaces Introduced:**
- `GovernanceService`

**Tests Required:**
- Projection logic
- All status combinations
- Refresh logic

**Acceptance Criteria:**
- Governance projection correctly reflects Solvent state
- Blockers accurately identified
- Next task logic works

**Dependencies:** Phase 3

**Non-Goals:** No UI

---

### Phase 5: Minimal UI

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

**Dependencies:** Phase 4

**Non-Goals:** No complex frontend framework, no real-time updates (use polling)

---

### Phase 6: Agent Integration Test with Stub

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

**Acceptance Criteria:** Stub completes synthetic task end-to-end

**Dependencies:** Phase 5

**Non-Goals:** No real agent integration

---

### Phase 7: End-to-End Synthetic SDLC Workflow

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

**Dependencies:** Phase 6

**Non-Goals:** No theatrical demo, no manual predetermination

---

### Phase 8: Readiness Gate for Oracle

**Objective:** Verify scaffolding supports domain-specific extensions.

**Scope:**
- Document Oracle-specific mappings
- Verify all Oracle concepts map to existing entities
- Identify gaps

**Files/Packages Introduced:** None

**Schema Changes:** None

**Interfaces Introduced:** None

**Tests Required:** Oracle readiness checklist

**Acceptance Criteria:**
- Oracle can be implemented as a project within this system
- No new kernel primitives needed
- No new project-layer entities needed

**Dependencies:** Phase 7

**Non-Goals:** No Oracle implementation

---

## 15. Oracle Readiness

### Oracle Concepts Mapped to Conductor

| Oracle Concept | Conductor Mapping |
|----------------|-------------------|
| Explorer | Agent identity |
| Attacker | Agent identity (different agent) |
| Verifier / Judge | Agent identity (reviewer agent) |
| Human Reviewer | Human actor in Activity |
| Claim | Task with description containing claim |
| Evidence | Activity with action: "evidence_attached" |
| Artifact | Activity with action: "artifact_produced" |
| Verification work | Task with status: "active" |
| Unresolved obligation | Task with status: "blocked" |
| Reviewer | Task with current_agent set to reviewer |
| Decision | Activity with action: "decision_made" |
| Consequential action | Task with scenario_id set |

### What Oracle-Specific Behavior Belongs Above Conductor

- Domain-specific trust semantics
- Physics-specific verification logic
- Theorem-proving concepts
- Domain-specific evidence types
- Domain-specific authority models

### What Conductor Must Expose for Oracle

- Task lifecycle with governance intersection
- Agent coordination for multiple agent types
- Activity logging for audit trail
- Governance projection for Solvent state
- MCP tools for agent access

---

## 16. Security and Trust Boundaries

### Trust Boundaries

| Boundary | Trust Model |
|----------|-------------|
| Human → Conductor | Authenticated via API key (deployment boundary) |
| Agent → Conductor | Authenticated via API key or MCP trusted local |
| Conductor → Solvent | Read-only. Never writes. Never creates authority. |
| Solvent → Conductor | Solvent does not know Conductor exists. |
| Agent → Solvent | Via Solvent MCP. Agent creates beliefs, promotes, authorizes. |
| Agent → Executor | Via Solvent authority service. Agent cannot bypass. |

### Authentication

- **Human:** API key in HTTP header
- **Agent:** API key or MCP trusted local process
- **Conductor → Solvent:** API key to Solvent API

### Attribution

- All activities recorded with actor_type and actor_id
- Agent activities attributed to agent_id
- Human activities attributed to human identifier
- System activities attributed to "system"

### Authorization

- Conductor manages project/task lifecycle
- Solvent manages authority and governance
- Conductor never decides "this is authorized"
- Solvent never decides "this task is complete"

### Responsibility

- **Conductor:** Project coordination, task lifecycle, activity logging
- **Solvent:** Authority, governance, consequential execution
- **Agent:** Performing work, making governance decisions
- **Human:** Genuine judgment, authorization approvals

### Trusted vs Untrusted Input

- **Trusted:** Conductor internal state transitions
- **Untrusted:** Agent-provided metadata, activity details
- **Validated:** Task status transitions, agent identity

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
- Oracle-specific domain logic
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
- Caching layer (beyond 5-second governance cache)
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
| **Core entities** | Project, Task, Activity (3 tables) | More entities | Minimal viable model |
| **Agent model** | String identifier on Task | Separate agent table | Simpler v0, extend later |
| **API** | REST + MCP | MCP only | Human and agent access |
| **MCP** | 11 tools | More tools | Minimal viable surface |
| **Solvent boundary** | Read-only adapter | Read-write | Preserves kernel freeze |
| **Governance projection** | Computed, never stored | Cached in DB | No duplicate truth |
| **UI** | Server-rendered HTML | SPA | Minimal complexity |
| **Execution boundary** | Agent + Solvent | Conductor executes | Clear responsibility |
| **Testing strategy** | Real SQLite + stub Solvent | Real Solvent for all | Deterministic tests |
| **Implementation sequence** | 8 phases | Different order | Logical dependency chain |
| **Major non-goals** | No Jira, no workflow engine, no Solvent changes | Various | Architectural focus |

---

## What Must Remain Outside Solvent

The following responsibilities belong OUTSIDE the frozen Solvent kernel:

1. **Project lifecycle management** — creating, updating, completing projects
2. **Task lifecycle management** — creating, assigning, updating, completing tasks
3. **Agent coordination** — assigning agents to tasks, tracking agent progress
4. **Activity logging for project events** — recording what agents and humans did
5. **Dependency tracking** — recording blocking relationships between tasks
6. **UI presentation** — rendering project state for humans
7. **Agent MCP tools for project operations** — giving agents access to project state
8. **Governance projection** — reading and projecting Solvent state for project tasks
9. **Workflow orchestration** — determining what happens next in the project lifecycle
10. **Specification management** — storing and versioning task descriptions

These are ALL project management concerns. Solvent governs trust, authority, and consequential execution. The project layer manages the workflow of getting work done. The two systems observe the same lifecycle without collapsing their responsibilities.

---

*This document was produced by inspecting the actual Solvent repository at `/home/chaschel/Documents/go/solvent-main/` and the Paca repository at `/home/chaschel/Documents/go/paca-master/`. No files were modified. No implementation was produced. This is a design specification awaiting review and approval.*
