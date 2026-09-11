CONDUCTOR — COMPREHENSIVE ADVERSARIAL REVIEW

You have now completed all planned Conductor phases (0–8).

Do NOT implement new features.
Do NOT redesign the architecture.
Do NOT modify Solvent.
Do NOT modify the frozen Solvent kernel.
Do NOT "fix" findings by expanding scope before reporting them.

Perform a rigorous adversarial review of the ACTUAL IMPLEMENTED CONDUCTOR REPOSITORY.

The purpose is to determine whether the implementation faithfully preserves the architecture we deliberately established, not merely whether the tests pass.

==================================================
LOCKED ARCHITECTURAL MODEL
==================================================

AI / Coding / Research Agent
    = AGENCY

Conductor
    = COORDINATION

Solvent
    = AUTHORITY

External Executor / External System
    = EFFECT

Hard invariant:

    CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION

Additional hard invariant:

    Every authoritative fact belongs to exactly one system.

Domain meaning is NOT a Conductor or Solvent infrastructure
responsibility.

Conductor is domain-agnostic.

Conductor is NOT:
- an agent runtime
- a reasoning engine
- a domain ontology engine
- a policy engine
- an authorization engine
- an execution engine
- a Solvent replacement

Solvent is NOT:
- a project manager
- a task manager
- an agent coordinator
- a domain reasoning engine
- a scientific judge
- an execution platform

==================================================
CONDUCTOR RESPONSIBILITY
==================================================

Conductor owns:

- Project
- Task
- Dependency
- Assignment
- Task lifecycle
- Work coordination
- Project activity/history
- Agent-facing API
- Agent-facing MCP
- Minimal human UI
- Read-only governance observation

Conductor may observe external governance state.

Conductor must NEVER:
- authorize a consequence
- mutate governance
- execute a consequence
- broker consequential execution
- persist authorization as authoritative truth
- recreate Solvent's authority model
- infer authority from task state
- infer execution from activity
- embed domain semantics

==================================================
SOLVENT RESPONSIBILITY
==================================================

Solvent owns:

- governance
- evidence/belief/debt where applicable
- authority
- exact state binding
- target
- intent
- authorization
- revocation
- claim
- governance/execution boundary

Solvent determines:

    "May this exact consequential action happen
     against this exact state?"

AUTHORIZE ≠ EXECUTE

External systems/executors produce the actual effect.

==================================================
REVIEW OBJECTIVE
==================================================

Do not perform a superficial code-quality review.

Try to BREAK THE ARCHITECTURE.

Assume a malicious, buggy, overly clever, or confused AI agent will
attempt to exploit every ambiguity in Conductor.

Look for places where implementation behavior differs from the intended
architectural contract even though normal tests pass.

==================================================
1. ARCHITECTURAL BOUNDARY AUDIT
==================================================

Inspect the entire repository.

For every:
- package
- type
- database table
- database field
- service
- repository
- API endpoint
- MCP tool
- middleware
- UI action
- external integration
- cache/projection
- lifecycle rule

determine which responsibility it belongs to:

AGENT
CONDUCTOR
SOLVENT
EXTERNAL EXECUTOR
DOMAIN APPLICATION
NONE / UNNECESSARY

Identify ANY overlap.

Pay particular attention to hidden duplication of:
- authority
- authorization
- execution state
- governance state
- domain state
- agent state

Reject designs where Conductor and Solvent can independently claim
authority over the same fact.

==================================================
2. DOMAIN-AGNOSTICITY AUDIT
==================================================

Determine whether Conductor can genuinely coordinate arbitrary
knowledge-intensive work without understanding the domain.

Search for hidden assumptions about:
- software development
- security
- mathematics
- BM-IST
- deployment
- experiments
- research
- infrastructure
- specific providers
- specific agent vendors

The core must not encode domain ontology.

Verify that:
- Task remains generic
- lifecycle remains generic
- Activity remains generic
- dependencies remain generic
- MCP remains generic
- API remains generic
- governance_ref remains provider-routable but provider-payload opaque

A domain-specific workload must be able to use Conductor without
changing the core model.

==================================================
3. LIFECYCLE STATE-MACHINE AUDIT
==================================================

Attempt to bypass the lifecycle through EVERY available path.

Verify:

proposed → active
active → review
review → accepted
review → active
blocked → active
and allowed cancellation paths

Verify forbidden transitions.

Attempt to reach terminal states through:
- generic PATCH
- repository methods
- services
- API
- MCP
- malformed input
- repeated requests
- concurrent requests
- stale state
- direct parameter manipulation

Verify:
- status cannot be directly mutated through generic update
- current_agent cannot be spoofed or arbitrarily reassigned
- terminal states are truly terminal
- rejection is an event/decision, not a durable state
- lifecycle validation is enforced below the API boundary

==================================================
4. TRANSACTIONAL INTEGRITY
==================================================

Verify that every state transition + corresponding Activity event
is atomic.

Try to produce:
- state changed, activity missing
- activity recorded, state unchanged
- partial commits
- duplicate activity
- duplicate transitions
- inconsistent timestamps

Verify rollback behavior when any part fails.

Verify task claiming is truly atomic under concurrent callers.

Attempt races between:
- two agents claiming the same task
- submit vs cancel
- accept vs reject
- block vs submit
- unblock vs cancel
- concurrent generic updates

Do not accept application-level locking where a database invariant or
transaction is required.

==================================================
5. IDENTITY / AUTHENTICATION AUDIT
==================================================

Assume an adversarial agent can control all request payload fields.

Verify that:
- actor identity comes from authenticated identity
- agent_id cannot be asserted by request metadata
- an agent cannot impersonate another agent
- an agent cannot impersonate a human
- an agent cannot alter activity attribution
- MCP and HTTP identity semantics remain coherent

Check whether any endpoint or MCP tool accidentally trusts:
- agent_id
- actor_id
- actor_type
- reviewer identity
- project identity
- role-like metadata

without validating the caller.

==================================================
6. AUTHORIZATION AUDIT
==================================================

Conductor is NOT an authorization engine.

Verify that no Conductor operation effectively says:

- task assigned → authorized
- task accepted → authorized
- governance status ready → authorized
- agent role → authorized
- activity says approved → authorized

Verify that CheckAuthorization is purely informational.

Verify its result is never:
- persisted as truth
- converted into a Conductor permission
- used to authorize execution
- used to mutate task state as though it were authority

==================================================
7. SOLVENT INTEGRATION AUDIT
==================================================

Inspect the Solvent adapter carefully.

Verify:
- Solvent-specific vocabulary remains contained within adapter code
- generic interfaces remain genuinely generic
- governance_ref provider routing is the only provider-aware behavior
- reference_id and metadata remain opaque to Conductor core
- no aggregate Solvent statistics are used as guessed authority state
- only authoritative reference-specific state is translated
- provider errors degrade safely
- stale/unknown states are represented honestly

Try to find any path where:
- Solvent state is reconstructed locally
- Conductor creates a shadow authority state
- Conductor interprets Solvent semantics
- a cached projection is mistaken for authority

==================================================
8. EXECUTION BOUNDARY AUDIT
==================================================

Prove that Conductor cannot execute consequential actions.

Search for:
- execution endpoints
- hidden execution helpers
- provider calls that produce side effects
- command execution triggered by Conductor
- webhooks
- shell execution
- external mutation paths
- "convenience" execution methods

The intended boundary is:

Agent
  → Conductor for work coordination

Agent
  → external governance system for consequential action

Conductor
  → read-only observation of governance

Verify there is no accidental path:

Agent → Conductor → execution

==================================================
9. ACTIVITY SEMANTICS AUDIT
==================================================

Verify Activity remains:
- append-only
- project/work history
- observational

Activity must NOT become:
- authority
- execution evidence
- artifact ownership
- domain truth

Attempt to exploit activity records such as:

"approved"
"executed"
"verified"
"proved"

to make Conductor infer stronger state.

Verify references/metadata are non-authoritative.

==================================================
10. GOVERNANCE PROJECTION AUDIT
==================================================

Verify governance projection is:
- read-only
- derived
- non-authoritative
- clearly separated from Conductor state

Check:
- freshness
- stale
- unknown
- provider unavailable
- malformed provider output
- inconsistent provider responses

Verify no stale projection can be mistaken for a current authorization.

Verify UI and API wording do not imply:

"Conductor authorized this."

==================================================
11. MCP ADVERSARIAL AUDIT
==================================================

Treat MCP as an attack surface.

Attempt:
- lifecycle bypass
- identity spoofing
- unauthorized task modification
- unauthorized acceptance/rejection
- cross-project dependency manipulation
- cross-agent impersonation
- governance mutation
- execution
- malformed arguments
- missing arguments
- repeated operations
- race conditions

Verify MCP cannot bypass invariants enforced by the underlying domain/store
layer.

==================================================
12. HTTP API ADVERSARIAL AUDIT
==================================================

Attempt the same attacks through HTTP.

Check:
- authentication
- authorization of project/task operations
- actor attribution
- field-level mutation
- lifecycle enforcement
- invalid transitions
- cross-project access
- malformed input
- replay/retry behavior
- concurrency

Generic update must not become a universal mutation backdoor.

==================================================
13. CROSS-PROJECT ISOLATION
==================================================

Verify task dependencies cannot cross project boundaries.

Try:
- creating dependency across projects
- updating dependency to another project
- deleting/reassigning project
- manipulating task IDs across projects
- using malformed identifiers

Verify referential integrity is preserved.

==================================================
14. DATA MODEL MINIMALITY
==================================================

Determine whether any implemented entity, field, table, endpoint, MCP tool,
or abstraction has accidentally become a responsibility that was explicitly
excluded from Conductor.

Look especially for creeping concepts such as:
- agent registry
- workflow engine
- policy engine
- authorization state
- execution state
- domain ontology
- artifact store
- notification system
- event bus
- hidden scheduler

Do not recommend additions merely because they are conventional.

The standard is the LOWEST COMMON DENOMINATOR.

==================================================
15. FAILURE / RECOVERY AUDIT
==================================================

Test or reason through:

- agent crash
- abandoned task
- duplicate request
- partial request
- database failure
- governance failure
- governance timeout
- stale governance
- malformed external response
- external execution failure
- ambiguous external outcome
- agent replacement
- reviewer failure

Ensure Conductor does not make claims stronger than the information it
actually owns.

==================================================
16. DOMAIN-WORKLOAD TEST
==================================================

Use BM-IST as one workload example, but do NOT redesign around BM-IST.

Also mentally substitute:
- Go software development
- cybersecurity
- scientific computing
- data engineering
- infrastructure operations
- another unrelated knowledge-intensive domain

Determine whether the same Conductor core remains sufficient.

The goal is not to prove that Conductor can perform the domain work.

The goal is to prove that Conductor can COORDINATE the work without
understanding its domain meaning.

==================================================
17. FOUR-WAY SEPARATION TEST
==================================================

For every important capability, classify it as exactly one of:

CAPABILITY
WORK
AUTHORITY
EXECUTION

Ask:

- Can an agent perform this?
- Is it officially assigned work?
- Is it authorized?
- Has it actually happened?

The implementation must never silently infer one from another.

Specifically prove:

capability      ≠ work
work            ≠ authority
authority       ≠ execution
activity        ≠ execution fact