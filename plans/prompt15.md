Draft the implementation plan for the Conductor repository's AGENTS.md.

IMPORTANT:
- This is a Conductor repository/documentation task.
- There is NO BM-IST code in this repository.
- Do NOT implement or design BM-IST inside Conductor.
- BM-IST is only a future/example workload that may validate Conductor later.
- The resulting AGENTS.md must remain completely domain-agnostic.
- Do NOT add BM-IST terminology, scientific concepts, research ontology, physics concepts, or domain-specific workflows to Conductor's architecture or implementation rules.

Use the attached technical writeup "Conductor + Solvent + AI Agents" as the architectural background, but adapt it specifically into repository-level engineering guidance for Conductor.

The goal is to produce a DRAFT IMPLEMENTATION PLAN for AGENTS.md first.
Do NOT write the final AGENTS.md yet.
Do NOT implement code.

==================================================
LOCKED ARCHITECTURAL BASELINE
==================================================

AI / Coding / Research Agent = AGENCY
Conductor                  = COORDINATION
Solvent                    = AUTHORITY
External Executor          = EFFECT

Hard invariant:

CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION

Additional invariant:

DOMAIN MEANING is outside the Conductor infrastructure layer.

Conductor must remain:
- domain-agnostic
- a coordination substrate
- project/work oriented
- agent-facing
- read-only with respect to external governance
- independent of the Solvent database/kernel

Conductor is NOT:
- an agent runtime
- a reasoning engine
- a research engine
- a domain ontology
- a policy engine
- an authorization engine for consequential actions
- an execution engine
- a Solvent replacement

Solvent remains the authority system.

External executors create external effects.

==================================================
CURRENT CONDUCTOR MODEL
==================================================

The current generic Conductor model is intentionally small:

- Project
- Task
- Dependency
- Assignment
- Task lifecycle
- Activity
- Governance reference / read-only governance observation

The current validated responsibility is:

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

==================================================
RESPONSIBILITY BOUNDARY
==================================================

AGENT

The agent may:
- reason
- interpret
- propose
- investigate
- implement
- use tools
- produce artifacts
- report
- propose new work
- perform ordinary work
- request consequential actions

But:

agent capability ≠ work assignment
agent capability ≠ authority

CONDUCTOR

Conductor owns:
- project/work state
- tasks
- dependencies
- assignments
- lifecycle
- project activity
- agent-facing API/MCP
- minimal human situational-awareness UI
- read-only governance observation

SOLVENT

Solvent owns:
- authority
- authorization
- exact target/state binding
- intent
- revocation
- claim
- consequential authorization boundary

EXECUTOR

External executors own:
- actual external effects
- actual external execution outcomes

==================================================
NON-OVERLAP RULE
==================================================

Every authoritative fact must have one owner.

Conductor must not create:
- shadow authorization state
- shadow execution state
- domain truth
- duplicate Solvent authority state

Solvent must not own:
- projects
- tasks
- dependencies
- assignment
- Conductor lifecycle
- project activity

Domain applications/workloads must not require Conductor changes merely
because they have different domain semantics.

==================================================
CONDUCTOR / SOLVENT BOUNDARY
==================================================

Conductor may observe external governance through a generic read-only
GovernanceReader abstraction.

The governance reference is provider-routable but provider-specific payload
remains opaque to Conductor.

Conductor must never:
- authorize a consequence
- execute a consequence
- mutate Solvent state
- persist Solvent authorization as Conductor truth
- infer authorization from task state
- infer execution from Activity
- interpret provider-specific governance semantics in the generic core

Solvent must never become a dependency of ordinary Conductor task
coordination.

Ordinary work coordination must remain functional even when external
governance is unavailable, except where the domain/application explicitly
requires governance for a consequential step.

==================================================
DOMAIN AGNOSTICITY
==================================================

AGENTS.md must explicitly prevent domain leakage.

Do NOT encode:
- BM-IST
- physics
- mathematics
- software development
- cybersecurity
- scientific workflows
- experiments
- research ontology
- proofs
- hypotheses
- evidence semantics
- domain-specific policy
- domain-specific task states

Use only generic examples such as:
- task
- artifact
- dependency
- external action
- governed consequence
- project work

The same Conductor core should remain usable for:
- software engineering
- scientific work
- cybersecurity
- data engineering
- infrastructure operations
- other knowledge-intensive domains

without changing the core model.

==================================================
IMPLEMENTATION PRINCIPLES
==================================================

AGENTS.md should establish:

1. Minimality
   Add the smallest generic primitive necessary.
   Do not add a feature merely because a particular workload wants it.

2. Lower-layer invariants
   Important Conductor invariants must be enforced below the API/MCP
   surface so callers cannot bypass them.

3. Lifecycle integrity
   Generic update operations cannot bypass lifecycle transitions.

4. Atomicity
   Lifecycle state changes and required Activity records must remain atomic.

5. Identity
   Actor identity must derive from an authenticated/trusted boundary,
   never from arbitrary request metadata.

6. Agent untrustedness
   Treat autonomous agents as callers that may be buggy or adversarial.
   Do not trust claims merely because they came from an agent.

7. Activity semantics
   Activity is project/work history, not authority or proof of external
   execution.

8. Governance projection
   Governance observation is non-authoritative Conductor state.
   Unknown/unavailable must not become authorization.

9. Execution separation
   Conductor never becomes an execution broker.

10. Adapter boundary
    External systems belong behind adapters/services rather than being
    embedded into the generic domain model.

11. No kernel/application confusion
    Keep Conductor's generic coordination model separate from external
    governance and domain applications.

12. Ecosystem over core growth
    Prefer adapters, integrations, services, policies, and deployment
    boundaries over adding new Conductor core concepts.

==================================================
IMPLEMENTATION HISTORY / CURRENT STATE
==================================================

The repository has already gone through:
- repository/architecture foundation
- domain model + SQLite persistence
- API/MCP
- governance observation
- minimal UI
- agent integration
- end-to-end coordination validation
- domain extension validation
- adversarial review and remediation

The coding agent should inspect the ACTUAL REPOSITORY before writing AGENTS.md.

Do not assume documentation is more authoritative than code.

Reconcile:
- package boundaries
- actual APIs
- current repository structure
- current lifecycle
- current database model
- governance adapter boundary
- authentication boundary
- MCP boundary
- UI boundary
- current tests

If there is a discrepancy between the architectural principles above and
the actual repository implementation, identify it in the plan rather than
silently rewriting the architecture.

==================================================
PURPOSE OF AGENTS.md
==================================================

AGENTS.md should function as a durable repository-level engineering contract
for future AI coding agents and human contributors.

It should answer:

- What is Conductor?
- What is Conductor responsible for?
- What is explicitly outside Conductor?
- How is Conductor different from Solvent?
- How should agents interact with Conductor?
- What invariants must never be bypassed?
- What kinds of changes require architectural review?
- What should be implemented outside the core?
- What must remain domain-agnostic?
- How should external integrations be structured?
- What does "done" mean for a Conductor implementation?

It should prevent future agents from gradually turning Conductor into:
- Jira/Linear clone
- agent framework
- workflow engine
- research engine
- policy engine
- authorization engine
- execution engine
- domain-specific application

==================================================
REQUIRED PLAN OUTPUT
==================================================

Produce an implementation plan for AGENTS.md with these sections:

1. Purpose and status
2. Conductor architectural thesis
3. Agency / Coordination / Authority / Effect model
4. Capability / Work / Authority / Execution invariant
5. Conductor responsibility boundary
6. Explicit non-responsibilities
7. Agent interaction model
8. Solvent integration boundary
9. External executor boundary
10. Domain-agnosticity rules
11. Core data/lifecycle invariants
12. Authentication / identity rules
13. API and MCP engineering rules
14. Database / transaction rules
15. Adapter/integration rules
16. Testing and verification expectations
17. Architectural change / growth gate
18. Common anti-patterns
19. Definition of done
20. Repository-specific implementation guidance

For every proposed AGENTS.md rule, classify it as:

- MUST
- SHOULD
- MAY
- MUST NOT

Prefer normative statements over prose explanations.

==================================================
ARCHITECTURAL GROWTH GATE
==================================================

Include a rule equivalent to:

A new Conductor core primitive requires evidence that the requirement is a
generic coordination invariant that cannot be expressed through the
existing coordination model or an external adapter/service.

The fact that one domain workload needs a feature is NOT sufficient.

Likewise, any proposed Solvent change must remain subject to the separate
Solvent kernel-growth discipline.

==================================================
IMPORTANT CONSTRAINTS
==================================================

Do NOT:
- implement code
- modify Solvent
- add BM-IST concepts
- propose a BM-IST architecture
- create a research engine
- add project-management features merely because conventional PM tools have them
- introduce RBAC/multi-tenancy unless the repository already requires it
- expand Conductor's authority boundary
- make Conductor part of consequential execution

Do:
- inspect the actual Conductor repository
- reconcile documentation with implementation
- preserve the smallest generic model
- make boundaries explicit
- make future AI coding agents less likely to cause architectural drift

Finally output:

A. Proposed AGENTS.md structure
B. Normative rules to include
C. Repository-specific facts that should be encoded
D. Architectural anti-patterns to prohibit
E. Open ambiguities requiring human decision
F. Final recommendation for whether the plan is ready to write AGENTS.md

Do not write AGENTS.md yet.
Stop after the implementation plan.