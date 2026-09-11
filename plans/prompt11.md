CONDUCTOR — FOCUSED REMEDIATION

The comprehensive adversarial review identified implementation defects that
must be corrected before Conductor is considered validated for real use.

DO NOT redesign the architecture.
DO NOT add new architectural layers.
DO NOT modify the Solvent repository.
DO NOT modify the frozen Solvent kernel.
DO NOT add domain-specific concepts.
DO NOT turn Conductor into an authorization engine, agent framework,
workflow engine, or execution platform.

Preserve the locked architecture:

    AI / Coding / Research Agent = AGENCY
    Conductor                  = COORDINATION
    Solvent                    = AUTHORITY
    External Executor          = EFFECT

    CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION

And:

    Every authoritative fact belongs to exactly one system.

==================================================
P0 — MUST FIX
==================================================

1. WIRE HTTP AUTHENTICATION

The existing Auth middleware is not currently applied to the HTTP router.

Fix the HTTP server so every protected /v1/* operation runs through the
authentication boundary.

Requirements:
- derive actor identity from authenticated credentials
- never trust agent_id / actor_id / actor_type supplied in request bodies
- reject missing/invalid credentials
- preserve the existing Conductor resource model
- do not introduce a new authorization engine

Authentication establishes WHO is calling Conductor.
It does not determine consequential authority.

Tests:
- unauthenticated request rejected
- invalid credential rejected
- authenticated request receives derived actor identity
- caller-supplied actor identity cannot override authenticated identity

2. PREVENT TASK CREATION LIFECYCLE BYPASS

Task creation must not permit callers to establish lifecycle or assignment
state directly.

Enforce:

    new task:
        status = proposed
        current_agent = NULL

Generic task creation/update must NOT directly establish:
- active
- accepted
- cancelled
- blocked
- current_agent

Lifecycle fields may only change through the validated lifecycle operations.

Ensure this is enforced below the HTTP/MCP layer so no caller can bypass it.

Tests:
- create with current_agent rejected or ignored safely
- create with non-proposed status rejected or normalized safely
- task always enters proposed/unassigned state
- claim is the only path to assignment/active state

Prefer rejecting invalid caller input over silently accepting contradictory
state.

3. FIX AND WIRE SOLVENT GETSTATE

The production GovernanceReader path currently returns an unconditional
unknown state instead of retrieving the authoritative reference-specific
Solvent state.

Fix the actual production path.

Requirements:
- GovernanceReader.GetState must reach the configured Solvent adapter
- SolventAdapter must query the authoritative reference-specific Solvent
  endpoint
- translate the authoritative response into generic GovernanceState
- do not infer state from aggregate Solvent counters
- do not invent governance semantics in Conductor
- do not persist the result as Conductor truth

The implementation must use the same path that the HTTP governance endpoint,
MCP governance tool, and UI governance projection use.

Tests must prove:
- a known Solvent state is returned correctly
- unknown/unavailable Solvent is distinguishable
- malformed provider responses fail safely
- no aggregate statistics are used for status inference

4. WIRE GOVERNANCESERVICE / PROVIDER ROUTING

The implemented GovernanceService/provider-routing path must actually be used
by production request paths.

Do not leave provider routing as dead infrastructure.

Requirements:
- configure registered GovernanceReader implementations
- route governance_ref.provider to the correct reader
- use NullReader only when explicitly configured/defaulted
- avoid direct bypasses from API/web to a hard-coded reader

Tests:
- provider routing reaches the expected reader
- unknown provider handled safely
- default/no-provider behavior is explicit
- Solvent provider reaches SolventAdapter

5. PROTECT governance_ref FROM ARBITRARY MUTATION

An authenticated agent must not be able to silently redirect an existing task
from one governance context/provider to another.

Preferred v1 rule:

    governance_ref is write-once at task creation.

After creation it is immutable.

Do not add complex governance policy to Conductor.

Requirements:
- create_task may establish governance_ref
- generic update_task may NOT modify governance_ref
- lifecycle operations may NOT modify governance_ref
- MCP and HTTP must enforce the same rule
- repository/service layer must enforce it, not only handlers

Tests:
- legitimate creation with governance_ref works
- later mutation is rejected
- HTTP cannot mutate it
- MCP cannot mutate it
- repository/service cannot mutate it through ordinary update

==================================================
P1 — MUST FIX BEFORE REAL POC USE
==================================================

6. FIX dependency-aware next_task

Conductor is a coordination system.

next_task must not return a task whose dependencies are unresolved.

For a task to be eligible:
- status = proposed
- current_agent = NULL
- all same-project dependencies are resolved according to the existing
  dependency semantics

Do not add scheduling heuristics, priorities, capability matching, or an
AI planner.

Implement the minimum deterministic dependency check.

Tests:
- blocked task is skipped
- eligible task is returned
- multiple eligible tasks handled deterministically
- completed/accepted dependency permits dependent work
- cancelled/blocked dependency follows the documented semantics

7. ENFORCE TASK/PROJECT ACCESS BOUNDARIES

A caller must not be able to retrieve arbitrary task records merely by
guessing a task ID when they do not have access to that project's context.

Do not introduce complex RBAC.

At minimum:
- authenticate caller
- establish project/task access according to the minimal Conductor model
- ensure direct task retrieval respects project scope

Review:
- get_project
- list_tasks
- get_task
- update_task
- claim_task
- submit_task
- accept_task
- reject_task
- blocker operations
- post_activity
- governance reads

Tests must prove cross-project task access is rejected.

8. PROVIDE MINIMAL ABANDONED-TASK RECOVERY

A claimed task can become stuck if the agent crashes or disappears.

Expose the already-existing release/reassignment capability through the
proper public Conductor surface, using authenticated actor semantics.

Do NOT build:
- heartbeat system
- distributed lease service
- scheduler
- agent registry

Use the smallest mechanism that allows legitimate recovery.

Tests:
- abandoned task can be released/reassigned according to defined rules
- unauthorized actor cannot release another agent's task
- activity event records the recovery
- lifecycle remains valid

9. REPLACE FAKE UUID GENERATION

Replace timestamp-derived IDs pretending to be UUIDs with a standard collision-
resistant identifier implementation.

Apply consistently to:
- project IDs where applicable
- task IDs
- activity IDs
- dependency IDs

Tests should verify uniqueness assumptions and valid identifier generation.

==================================================
P2 — REVIEW / TIGHTEN, DO NOT OVERENGINEER
==================================================

10. ACTIVITY ACTION SEMANTICS

Do not turn Activity into a second authority or execution record.

Keep Activity append-only and observational.

Review whether arbitrary action strings can create misleading semantics such
as:

    deploy.completed
    execution.finished
    verified
    approved
    proved

Choose the smallest improvement.

Possible acceptable approach:
- preserve generic activity semantics
- document that action is agent/project event metadata
- optionally restrict known lifecycle events while allowing generic activity

Do NOT create a domain event ontology.

Tests should prove activity cannot alter:
- task authority
- governance state
- execution state

==================================================
DO NOT IMPLEMENT THESE
==================================================

Explicitly do NOT add:

- Conductor authorization engine
- Conductor authority model
- capability registry/matching
- Solvent kernel changes
- Solvent database access
- Solvent mutation through Conductor
- execution endpoints
- execution broker
- local MCP authentication beyond the established trusted-local
  deployment boundary
- agent swarm framework
- general workflow engine
- scheduler
- heartbeat infrastructure
- domain-specific task states
- BM-IST concepts
- research ontology
- belief/evidence/debt tables
- artifact store
- new Conductor tables unless one is strictly required by the existing
  remediation and cannot be represented through the current model
- new Solvent primitives

IMPORTANT:

Do NOT "fix" the absence of Conductor authorization by adding a second
authorization system.

Conductor needs authenticated access control for its OWN resources and
operations.

That is distinct from Solvent authority.

    Conductor access control ≠ Solvent authority

==================================================
ARCHITECTURAL BOUNDARY CHECK AFTER REMEDIATION
==================================================

After making the fixes, verify this exact model still holds:

AGENT
  reason / propose / act / report
        │
        ▼
CONDUCTOR
  project / task / dependency / assignment / lifecycle / activity
        │
        │ read-only governance observation
        ▼
GOVERNANCEREADER
        │
        ▼
SOLVENT
  authority / target / state binding / authorization / revocation / claim
        │
        ▼
EXTERNAL EXECUTOR
  actual external effect

The Agent may interact directly with Solvent/external systems for
consequential actions.

Conductor must never become part of that execution path.

==================================================
VERIFICATION REQUIREMENTS
==================================================

After remediation run the complete test suite plus targeted adversarial
tests for every fixed finding.

At minimum verify:

1. HTTP authentication actually executes in the live router
2. MCP trusted-local behavior remains consistent with the deployment boundary
3. task creation cannot establish assignment/lifecycle state
4. generic task update cannot mutate lifecycle fields
5. governance_ref cannot be redirected after creation
6. real Solvent reference-specific state reaches production GetGovernance path
7. GovernanceService provider routing is actually wired
8. no aggregate Solvent statistics are used for inference
9. next_task respects dependencies
10. cross-project task access is blocked
11. abandoned tasks are recoverable through supported operations
12. identifiers are collision-resistant
13. state transition + activity remain atomic
14. activity remains append-only and non-authoritative
15. Conductor contains no execution path
16. Conductor contains no competing authority state
17. no domain-specific concepts entered the core

==================================================
OUTPUT
==================================================

Do not perform another architecture redesign.

Produce:

# Conductor Remediation Report

## 1. Changes Made
For every finding:
- finding number
- change
- files changed
- reason

## 2. Tests Added / Updated
List the tests and what attack/failure they prove.

## 3. Architecture Conformance
Explicitly state whether:
- Agent = Agency
- Conductor = Coordination
- Solvent = Authority
- Executor = Effect

remain intact.

## 4. Remaining Findings
Only genuine unresolved issues.

Classify:
- CRITICAL
- HIGH
- MEDIUM
- LOW
- INFO

Do not manufacture findings.

## 5. Final Verdict

Return exactly one:

GREEN — REMEDIATION COMPLETE

YELLOW — REMEDIATION INCOMPLETE

RED — ARCHITECTURAL BOUNDARY VIOLATED

GREEN requires:
- no CRITICAL findings
- no HIGH findings
- no lifecycle bypass
- no identity/authentication bypass
- no competing Conductor/Solvent authority
- no execution path through Conductor
- functional reference-specific Solvent observation
- domain-agnostic core preserved

Do not modify Solvent.
Do not implement unrelated improvements.
Stop after the remediation report.