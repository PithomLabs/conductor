Yes. At this point I would **not approve implementation yet**. The next step should be to have the coding agent produce a **written implementation plan** that reconciles the architecture we have agreed on.

Use this as the prompt:

---

## Coding Agent Prompt — Produce the Written Implementation Plan

We have agreed on the following architectural direction for the new project-layer system, currently named **Conductor**.

### Architectural decisions already made

**Conductor is a standalone project layer.**

It must be developed as a **separate repository and separate binary/process** from Solvent.

Do **not** modify the Solvent repository.

Do **not** add project-layer tables to Solvent's CockroachDB.

Do **not** modify Solvent kernel code, kernel schema, migrations, invariants, or APIs unless an existing public interface genuinely cannot support the integration. The Solvent kernel is considered **frozen**.

Conductor owns its own **standalone SQLite database**.

The boundary is:

```text
                 CONDUCTOR
        ┌─────────────────────────┐
        │ Project lifecycle       │
        │ Task lifecycle          │
        │ Agent coordination      │
        │ Activity history        │
        │ Workflow orchestration  │
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

Conductor must **never directly read or write Solvent's database**.

Conductor must **never become a second authority engine**.

Solvent remains authoritative for governance/authorization state.

Conductor remains authoritative for project/work state.

The project should demonstrate that an agent-native SDLC coordination layer can be built independently and can use Solvent as a governance boundary.

---

# Objective

Produce a **complete written implementation plan only**.

Do **not write implementation code yet**.

Do **not begin Phase 0 automatically**.

The purpose of this document is to let us review and approve the architecture before implementation begins.

The plan should be concrete enough that, after approval, another coding agent could execute it phase-by-phase without having to redesign the architecture.

---

# 1. Repository Architecture

Define the proposed standalone Conductor repository structure.

Show:

```text
repository/
├── ...
```

Explain the responsibility of each major directory/package.

Keep the architecture intentionally small.

Avoid introducing frameworks, abstractions, or infrastructure that are not justified by the project.

The repository should have an independently buildable binary such as:

```text
cmd/conductor/
```

and must have its own:

```text
go.mod
SQLite database
configuration
tests
HTTP/API boundary
MCP boundary
```

Do not assume Conductor will remain permanently in the Solvent repository.

Design the boundary so the two projects can evolve independently.

---

# 2. Domain Model

The current proposal contains three core tables:

```text
Project
Task
Activity
```

Review this model critically.

Determine whether these three entities are sufficient for the first implementation.

Do not add tables merely because they are conventional project-management features.

For every proposed entity, explain:

* why it exists
* what it owns
* what it must not own
* why it belongs in Conductor rather than Solvent

Explicitly evaluate whether concepts such as:

* specification
* agent assignment
* artifact
* dependency
* workflow
* governance projection

should be separate persistent entities or represented more simply.

Favor the smallest coherent model.

---

# 3. SQLite Design

Design the initial SQLite schema.

For every table specify:

* columns
* types
* primary keys
* foreign keys
* indexes
* uniqueness constraints
* lifecycle/status fields
* timestamps
* append-only requirements where applicable

Explain important invariants.

In particular, distinguish:

```text
Project state
Task state
Activity history
Governance projection
```

from Solvent's authoritative state.

Do not replicate Solvent's authority model into SQLite.

The Conductor database must remain a **coordination database**, not a shadow Solvent database.

---

# 4. Project and Task Lifecycle

Define the minimal lifecycle.

For example, evaluate a model such as:

```text
PLANNED
   ↓
ACTIVE
   ↓
BLOCKED / REVIEW
   ↓
DONE
```

Do not blindly accept this example.

Choose the smallest state machine that supports the intended agent-native SDLC workflow.

Document:

* allowed transitions
* invalid transitions
* who/what may perform each transition
* whether agents can perform them
* what events are recorded
* how failures are represented

The lifecycle must work for both human and AI agents.

---

# 5. Agent Model

Design the agent abstraction.

Conductor must be **agent-agnostic**.

Bob is the first integration target, but the domain model must not encode Bob-specific assumptions.

Define how Conductor represents:

```text
agent identity
agent assignment
agent status
current task
agent activity
agent-produced artifacts/results
```

Separate:

```text
project-layer agent identity
```

from:

```text
Solvent principal identity
```

Do not silently equate the two.

Explain how another coding agent could replace Bob later without changing the core domain model.

---

# 6. Agent API and MCP

Design the minimal API and MCP tool surface required for an agent to operate Conductor.

Prioritize actual workflow operations over a large API.

Consider capabilities such as:

```text
create/update project
create/update task
claim task
report progress
report blocked
submit result
attach artifact
request review
record activity
inspect governance status
```

These are examples, not requirements.

Determine the actual minimal set.

For every operation specify:

* purpose
* actor
* inputs
* outputs
* state mutations
* activity event generated
* whether it can trigger Solvent interaction

Do not create MCP tools simply because MCP permits them.

---

# 7. Solvent Adapter

Design the Conductor → Solvent integration boundary.

This is one of the most important parts of the plan.

Conductor must interact with Solvent through a narrow adapter/interface.

For example:

```text
Conductor
    ↓
SolventAdapter
    ↓
Solvent API / MCP / supported interface
```

The exact mechanism should be determined by examining the existing Solvent public interfaces.

The adapter must be **read-only from Conductor's perspective** unless an existing, already-approved Solvent operation explicitly requires otherwise.

The plan must specify:

* interface
* methods
* input/output structures
* error handling
* timeout behavior
* unavailable-Solvent behavior
* stale-data behavior
* caching rules, if any

Most importantly:

> Conductor must not infer or recreate authorization decisions that belong to Solvent.

---

# 8. Governance Projection

Define how Conductor presents Solvent governance state to humans and agents.

The projection may include concepts such as:

```text
Evidence
Belief
Debt
Authority
Intent
Execution
```

but these must remain **projections of Solvent state**, not duplicate domain concepts.

Specify:

* what Conductor reads
* how it maps the information
* when it refreshes
* whether it caches
* how stale information is marked
* what happens when Solvent is unavailable
* what happens when governance state changes while an agent is working

Clearly distinguish:

```text
displayed/projected governance state
```

from:

```text
authoritative Solvent state
```

---

# 9. Consequential Action Boundary

Explain exactly where the project workflow crosses from ordinary project coordination into consequential execution.

The desired conceptual flow is:

```text
Idea
 → Task
 → Agent work
 → Evidence
 → Belief
 → Governance readiness
 → Authority
 → Intent
 → Authorization
 → Consequential execution
 → Outcome
 → Project activity
```

Do not collapse Conductor's workflow state and Solvent's governance lifecycle into a single state machine.

Explain how the two lifecycles interact while remaining independently owned.

---

# 10. Minimal UI

Design a minimal server-rendered UI.

The UI exists for **human situational awareness and genuine human judgment**, not to reproduce Jira.

Determine the smallest set of useful views.

At minimum evaluate:

```text
Project overview
Task/work view
Activity stream
Governance status
```

Avoid unnecessary dashboards, charts, configuration screens, notifications, user-management systems, etc.

Explain how a human can understand:

```text
What is being worked on?
Who/what is doing it?
What happened?
What is blocked?
What does governance currently allow?
Why can't the next consequential step happen?
```

---

# 11. End-to-End SDLC Demonstration

Design one synthetic but realistic end-to-end workflow demonstrating Conductor.

It should cover a meaningful portion of:

```text
Idea
→ Requirement
→ Design
→ Implementation
→ Test
→ Review
→ Release
→ Deploy
```

and intersect with Solvent where authorization becomes consequential.

The workflow should demonstrate genuine coordination between:

```text
Human
Conductor
Coding Agent
Solvent
Execution system
```

Avoid constructing a theatrical demo where every result is manually predetermined.

The goal is for the architecture itself to make the demonstration compelling.

---

# 12. Failure and Race Conditions

Explicitly design for failure.

At minimum analyze:

```text
agent crashes
agent reports false completion
agent abandons task
two agents claim the same task
Conductor loses connection to Solvent
Solvent state changes while agent is working
authority becomes stale
authority is revoked
wrong target is requested
execution fails
execution result is ambiguous
project database is unavailable
agent is replaced
```

For each case state:

```text
source of truth
expected Conductor state
expected Solvent state
recovery behavior
human-visible result
```

Do not invent additional kernel mechanisms in Solvent to solve these problems.

---

# 13. Testing Strategy

Define the test strategy before implementation.

Include:

```text
unit tests
repository/database tests
HTTP/API tests
MCP tests
Solvent adapter tests
governance projection tests
agent integration tests
end-to-end workflow test
failure/race tests
```

Identify which tests use:

```text
real SQLite
Solvent test instance
stub Solvent adapter
fake agent
real external execution provider
```

The architecture should support deterministic tests without requiring the full external environment for every test.

---

# 14. Implementation Phases

Review and refine the current phases:

```text
Phase 0  Repository audit and architecture reconciliation
Phase 1  Project domain model and persistence
Phase 2  Agent API / MCP interface
Phase 3  Solvent integration adapter
Phase 4  Governance projection
Phase 5  Minimal UI
Phase 6  Agent integration test with stub
Phase 7  End-to-end synthetic SDLC workflow
Phase 8  Readiness gate for Oracle
```

For every phase provide:

```text
objective
scope
files/packages introduced
schema changes
interfaces introduced
tests required
acceptance criteria
dependencies
explicit non-goals
```

Also identify whether any phases should be reordered, merged, or split.

Do not preserve the existing phase structure merely for consistency.

---

# 15. Oracle Readiness

The eventual goal is for **Oracle / Trust-but-Verify** to become the first serious domain-specific project built on this architecture.

Do not implement Oracle in this project yet.

Instead, determine what Conductor must expose so that Oracle can later become a real project/workflow within the system.

The architecture should support:

```text
Explorer
Attacker
Verifier / Judge
Human Reviewer
```

without embedding Oracle-specific semantics into Conductor.

Explain what Oracle-specific behavior belongs above Conductor.

---

# 16. Security and Trust Boundaries

Document the trust boundaries.

At minimum:

```text
Human
Agent
Conductor
Solvent
Execution Provider
```

Explain:

* authentication
* attribution
* authorization
* responsibility
* trusted vs untrusted input
* which system is authoritative for which fact

Do not let Conductor's project-layer identity model accidentally become Solvent authority.

---

# 17. Explicit Non-Goals

Create a clear list of things that must **not** be implemented in the first version.

Examples to evaluate:

```text
full Jira replacement
multi-tenancy
complex RBAC
billing
notifications
chat platform
general workflow engine
plugin marketplace
agent swarm framework
Solvent kernel changes
Solvent schema changes
Solvent database access
Oracle-specific domain logic
large frontend framework
```

Add anything else that threatens the project's architectural focus.

---

# 18. Final Architecture Decision Record

Finish the document with a concise set of final decisions:

```text
Repository:
Binary:
Database:
Core entities:
Agent model:
API:
MCP:
Solvent boundary:
Governance projection:
UI:
Execution boundary:
Testing strategy:
Implementation sequence:
Major non-goals:
```

For every significant decision, include the alternative considered and why it was rejected.

---

## Critical constraint

Do **not** start coding.

Do **not** modify Solvent.

Do **not** create the repository yet.

Do **not** implement the schema yet.

Produce the **written implementation plan only**, then stop and wait for architectural approval.

The quality bar is:

> Another senior engineer should be able to read this document and understand exactly what Conductor is, what it owns, what Solvent owns, how the two communicate, why the boundary exists, and how the implementation will proceed without reopening the frozen Solvent architecture.

---

I would use this as the **gate between architecture approval and implementation**. The most important thing now is to force the coding agent to resolve the potentially dangerous areas—especially the domain model, Solvent adapter contract, governance projection, and the two interacting lifecycles—before it starts generating code.
