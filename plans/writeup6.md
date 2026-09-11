# Conductor + Solvent + AI Agents
## Technical Rationale and Review Background for the BM-IST Trust-Verifier POC

**Purpose:** Provide a common technical and architectural background for AI agents and human engineers reviewing the Conductor–Solvent–Agent system used by the BM-IST Synthesis physics trust-verifier POC.

**Audience:** AI coding/research agents, senior engineers, architects, reviewers, and human operators.

**Status:** Architectural background / review baseline. This document is not an implementation plan and does not authorize architectural expansion.

---

# 1. Executive Summary

The Pithom Labs architecture separates autonomous knowledge work into distinct responsibilities:

```text
AI / Coding / Research Agent = AGENCY
Conductor                  = COORDINATION
Solvent                    = AUTHORITY
External Executor          = EFFECT
```

The strongest invariant is:

```text
CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION
```

A fifth concept must remain outside those infrastructure responsibilities:

```text
DOMAIN MEANING
```

Domain meaning includes the semantics of whatever the system is studying or building. In the current POC that includes BM-IST mathematical and physical meaning. In another workload it could be software architecture, cybersecurity semantics, data-model semantics, or scientific interpretation.

The architecture is deliberately **not** a serial pipeline in which every action must pass through all three components.

Instead:

- the agent continuously reasons, proposes, acts, and reports;
- Conductor coordinates project/work state;
- Solvent is consulted when work reaches a consequential authority boundary;
- an external executor produces the real-world effect.

The architecture is therefore best understood as **three orthogonal control responsibilities surrounding domain work**, plus an external effect boundary.

The practical goal is not to build a universal research engine or a universal project-management platform. It is to identify and implement the smallest generic substrate needed for autonomous, non-trivial knowledge-intensive work.

---

# 2. The Architectural Problem

AI agents are increasingly capable of carrying out complex work:

- understanding large bodies of information;
- writing and modifying code;
- performing mathematical reasoning;
- running experiments;
- using external tools;
- coordinating multi-step investigations;
- producing artifacts;
- initiating consequential actions.

Capability creates a problem that is easy to state but easy to get wrong architecturally:

> A system capable of doing something is not necessarily authorized to do it.

A second problem is equally important:

> Work that exists operationally is not the same thing as authority to create a consequence.

A third is:

> A record saying that something happened is not necessarily proof that it happened in the external world.

This produces four distinct concepts:

```text
CAPABILITY
    What an actor is capable of doing.

WORK
    What work is actually part of the coordinated project.

AUTHORITY
    What consequential action is currently permitted.

EXECUTION
    What actually happened in the external world.
```

The architecture deliberately keeps those concepts separate.

---

# 3. The Pithom Labs Control Model

## 3.1 AI / Coding / Research Agents — Agency

The agent is the source of autonomous intellectual and operational activity.

The lowest common denominator is:

```text
reason
interpret
propose
investigate
implement
act
produce
report
```

The agent may:

- understand domain material;
- formulate hypotheses;
- construct arguments;
- search literature;
- write code;
- run ordinary tools;
- analyze results;
- attack assumptions;
- generate candidate actions;
- propose new work;
- produce artifacts;
- request a consequential action;
- report an observed result.

The agent does **not** automatically gain authority from any of these capabilities.

Examples:

```text
Agent can modify code
    ≠
Agent is assigned that work

Agent can call a GPU tool
    ≠
Agent is authorized to consume shared GPU resources

Agent claims an experiment succeeded
    ≠
The experiment is established as an authoritative external fact
```

The agent is therefore an **actor**, not the system of record for authority.

---

# 4. Conductor — Coordination

Conductor is the generic work-control plane.

Its irreducible question is:

> **What work exists, who is doing it, what depends on what, and what is its current work state?**

Conductor owns:

```text
Project
Task
Dependency
Assignment
Lifecycle
Activity
```

This is intentionally small.

Conductor does not need to understand:

- what a hypothesis means;
- whether a theorem is true;
- whether a physical theory is valid;
- whether evidence is scientifically sufficient;
- whether a research direction is promising;
- what a domain-specific artifact means.

For example:

```text
Project:
    BM-IST Phase 2

Task:
    Investigate a candidate realization

Dependencies:
    Model construction → experiment

Assignment:
    Agent-X

Status:
    active
```

Conductor can coordinate this without understanding the mathematics.

## 4.1 Conductor's role

Conductor:

- makes work operationally explicit;
- assigns work;
- sequences work;
- tracks dependencies;
- tracks progress;
- records project activity;
- exposes an agent-facing API/MCP interface;
- provides minimal human situational awareness;
- optionally observes external governance state.

Conductor does not become:

- a reasoning engine;
- a research engine;
- a scientific judge;
- a policy engine;
- an authorization engine;
- an execution engine;
- a replacement for Solvent.

---

# 5. Solvent — Authority

Solvent has a deliberately narrower purpose.

Its fundamental question is:

> **May this exact consequential action happen against this exact state?**

Solvent owns the authority-side concepts needed to answer that question, including:

```text
target
state/snapshot binding
intent
authority
authorization
revocation
claim
```

The exact implementation is the frozen Solvent kernel and its supported interfaces.

The critical distinction is:

```text
AUTHORIZE ≠ EXECUTE
```

Solvent does not need to understand the domain meaning of the action.

For BM-IST, Solvent does not need to understand:

- T2;
- T4;
- PYV;
- Koopman theory;
- Fisher information;
- S1–S10;
- mathematical truth;
- physical interpretation.

It can instead see a precise consequential action and determine whether that exact action is authorized against the exact relevant state.

For example:

```text
Agent:
    "Run a 500 GPU-hour experiment."

Conductor:
    "This is Task 84; dependencies are satisfied and the task is assigned."

Solvent:
    "Is this exact consequence authorized against this exact target/state?"

Executor:
    "The GPU job actually ran."
```

The mathematical or scientific meaning of the experiment remains outside Solvent.

---

# 6. External Executor — Effect

The executor is where the external consequence actually occurs.

Possible executors include:

```text
local process
GPU cluster
proof/verification farm
CAS or simulation environment
shared repository
CI/CD system
cloud infrastructure
database
publication system
other external systems
```

The important architectural property is not the technology.

It is that the operation produces a state change outside the agent's purely ephemeral workspace.

Therefore:

```text
Solvent
    = decides whether the consequence is authorized

Executor
    = produces the consequence
```

This prevents Solvent from becoming an execution platform.

---

# 7. Domain Meaning Is Not an Infrastructure Layer

The architecture intentionally avoids a fourth infrastructure layer such as:

```text
Agent
Conductor
Solvent
Research Engine
```

That would confuse domain workload with generic infrastructure.

Instead:

```text
DOMAIN WORK
    ↓
AI / AGENT
    ↓
CONDUCTOR
    ↓
SOLVENT (when a consequence requires authority)
    ↓
EXECUTOR
```

The domain workload supplies meaning.

For the current POC, BM-IST provides:

- mathematical objects;
- hypotheses;
- proofs;
- attacks;
- experimental questions;
- physical interpretation;
- scientific status;
- domain-specific verification.

These are workload/application concepts.

They are not reasons to expand Conductor or Solvent.

A future workload could replace BM-IST with:

- Go software engineering;
- cybersecurity;
- data engineering;
- infrastructure operations;
- another scientific domain;
- financial analysis;
- compliance research.

The infrastructure responsibilities remain unchanged.

---

# 8. The Strongest Architectural Invariant

The key invariant is:

```text
CAPABILITY
    ≠
WORK
    ≠
AUTHORITY
    ≠
EXECUTION
```

This catches four common category errors.

## 8.1 Capability does not imply work

```text
Agent has repository-write capability
    ≠
Agent is assigned the repository task
```

## 8.2 Work does not imply authority

```text
Task is assigned
    ≠
Consequential action is authorized
```

## 8.3 Authority does not imply execution

```text
Solvent authorizes action
    ≠
The action actually happened
```

## 8.4 Activity does not imply execution

```text
Agent records "experiment completed"
    ≠
The external experiment actually completed
```

These distinctions should be treated as architectural tests.

---

# 9. Three Different Kinds of State

A useful way to understand the entire system is to separate three internal state spaces plus the external world.

## 9.1 Agent state

```text
cognitive / operational state
```

Examples:

- current reasoning context;
- hypotheses being considered;
- temporary calculations;
- candidate actions;
- tool output;
- intermediate work.

Much of this can remain ephemeral.

## 9.2 Conductor state

```text
work state
```

Examples:

- project;
- task;
- dependency;
- assignment;
- lifecycle;
- activity.

Conductor owns the operational history of the work.

## 9.3 Solvent state

```text
authority state
```

Examples:

- target;
- exact state binding;
- intent;
- authority;
- authorization;
- revocation;
- claim.

Solvent owns the authoritative governance state.

## 9.4 External world state

```text
consequence / effect state
```

Examples:

- GPU job actually ran;
- repository actually changed;
- infrastructure actually mutated;
- publication actually occurred.

The external executor is the source of truth for the actual effect.

---

# 10. Non-Overlap Rule

Every authoritative fact must have exactly one owner.

## Conductor owns

```text
project state
task state
dependency state
assignment
work lifecycle
project activity
```

## Solvent owns

```text
authority
authorization
target/state binding
intent
revocation
claim
```

## External systems own

```text
actual external effect
```

## Domain application/workload owns

```text
domain meaning
semantic interpretation
domain-specific truth/status
```

## Agent owns

The agent owns its own current reasoning and operational process, but an agent report is not automatically authoritative merely because the agent produced it.

---

# 11. What This Means for the BM-IST Synthesis

BM-IST is a deliberately difficult workload because it contains almost every category of work that stresses the architecture:

```text
reasoning
planning
hypothesis generation
proof attempts
counterexample search
literature analysis
coding
simulation
expensive computation
review
artifact production
new work discovery
potentially consequential actions
```

That makes it an excellent validation workload.

It should not, however, become a reason to add BM-IST-specific infrastructure to Conductor or Solvent.

---

# 12. BM-IST Responsibility Mapping

## 12.1 Domain reasoning — Agent / Domain workload

These remain in the domain workload:

```text
axiom interpretation
definition interpretation
hypothesis generation
claim construction
proof construction
attack/counterexample search
literature interpretation
physical interpretation
mathematical analysis
result interpretation
competing explanation analysis
```

The agent may produce all of these.

A domain verifier or human may determine their semantic status.

Conductor does not need to understand them.

Solvent does not need to understand them.

---

# 13. Research Concepts and Their Lowest Common Denominator

Many BM-IST concepts should not become infrastructure entities.

| BM-IST concept | Lowest common denominator | Infrastructure owner |
|---|---|---|
| Axiom | Domain proposition | Domain workload |
| Definition | Domain proposition | Domain workload |
| Hypothesis | Candidate proposition | Domain workload |
| Claim | Proposition/output | Domain workload |
| Theorem | Verified proposition | Domain verifier |
| Evidence | Supporting artifact | Domain workload |
| Proof | Verification artifact | Domain workload / verifier |
| Attack | Adversarial artifact | Domain workload |
| Counterexample | Evidence against claim | Domain workload |
| Debt | Unresolved obligation | Domain/application |
| Research question | Work objective | Agent proposes; Conductor coordinates |
| Experiment | Work activity and possibly consequence | Agent + Conductor; external executor when applicable |
| Research snapshot | Domain/work-state snapshot | Domain/application |
| Version | Revision identity | Domain/application / Conductor metadata |
| Publication | External consequence | Executor; Solvent if governed |

The essential principle is:

> Durable data is not automatically a new infrastructure primitive.

A PDF proof, Git commit, numerical result, or manuscript can be an artifact without becoming a Conductor or Solvent primitive.

---

# 14. Research Dependencies vs Work Dependencies

One distinction is particularly important.

```text
DOMAIN DEPENDENCY ≠ WORK DEPENDENCY
```

Example:

```text
Mathematical theorem X is logically needed to justify Y
```

is a domain dependency.

Conductor only needs to know whether some operational task must precede another:

```text
Task B depends on Task A
```

It does not need to understand the theorem.

The same applies to physical or scientific dependencies.

This is one of the mechanisms that keeps Conductor domain-agnostic.

---

# 15. Research Snapshot vs Solvent Snapshot

These must not be conflated.

A BM-IST research snapshot might mean:

```text
the accepted domain state at a point in the research program
```

A Solvent authority snapshot means:

```text
the exact state against which an authority decision was created
```

They can be linked, but they are not the same object.

Therefore:

```text
research-state-042
    ≠
Solvent authority snapshot S-17
```

Even if the latter was informed by the former.

This distinction prevents domain state from silently becoming authorization state.

---

# 16. A Typical BM-IST Workflow

A representative workflow is:

```text
1. Agent studies the current BM-IST synthesis.

2. Agent identifies an unresolved research question.

3. Agent proposes an experiment or analytical task.

4. Conductor creates/records the work.

5. Conductor assigns the work and tracks dependencies.

6. Agent writes code, performs reasoning, and runs ordinary tools.

7. Agent produces an artifact/result.

8. Conductor records the work activity.

9. If a consequential operation is needed, the agent reaches the
   external governance boundary.

10. Solvent evaluates the exact consequential intent.

11. If authorized, the external executor performs the effect.

12. The result returns to the agent/domain workload for interpretation.

13. Conductor records the operational work history.

14. Agent proposes the next work.
```

The system is therefore a loop, not a one-way pipeline.

---

# 17. Example: Ordinary Local Computation

An agent runs:

```text
python analyze_spectrum.py
```

inside its own working environment.

Normally:

```text
Agent → local tool
```

No Solvent involvement is required merely because the work is scientifically important.

Conductor may know that the agent is working on a task.

The existence of the task does not itself make the local computation an authorized consequence.

---

# 18. Example: Expensive Shared Computation

Suppose the agent determines that the next experiment requires 500 GPU-hours on shared infrastructure.

The distinction is:

```text
Agent
    = proposes and prepares the experiment

Conductor
    = coordinates the work and dependencies

Solvent
    = determines whether the exact consequence is authorized

Executor
    = consumes the shared GPU allocation and runs the experiment
```

The fact that the task is assigned does not automatically authorize the cost.

---

# 19. Example: Shared Repository Mutation

Editing an ordinary working branch may remain ordinary agent activity, depending on the environment.

A protected production merge is materially different.

Conceptually:

```text
Agent
    → prepares change

Conductor
    → tracks work/review

Solvent
    → governs the consequential action if the environment requires it

Git / CI executor
    → performs the actual mutation
```

The exact policy is a domain/deployment concern, not something Conductor should hard-code.

---

# 20. Example: Publication

An agent may produce a manuscript as ordinary work.

The manuscript's scientific meaning remains a domain question.

If external publication itself is a governed consequence:

```text
Agent
    → produces manuscript

Conductor
    → coordinates review

Solvent
    → authorizes the specific publication consequence

Publication executor
    → publishes it
```

Again:

```text
publication_authorized
    ≠
publication_happened
```

---

# 21. The Role of the Physics Trust-Verifier

The BM-IST trust-verifier is a **domain workload/application**, not a fourth infrastructure system.

Its job is to supply domain-specific meaning and verification.

For example, it may determine:

```text
candidate claim
candidate evidence
proof status
counterexample status
research obligation
physical consistency
mathematical validity
```

Those judgments belong to the domain workload, its verifiers, or human reviewers.

Conductor coordinates the work surrounding those judgments.

Solvent governs only the consequential actions that require authority.

This distinction is crucial.

The verifier should not be implemented as:

```text
another authorization engine
```

and Conductor should not be turned into:

```text
another scientific knowledge graph
```

---

# 22. What the AI Agent Should Be Allowed to Do

The architecture intentionally gives the agent broad agency.

It may:

```text
read
reason
write code
run ordinary tools
generate artifacts
propose work
claim assigned work
report progress
submit work
request consequential actions
interpret results
propose the next step
```

The important limitation is not to cripple capability.

It is to prevent:

```text
capability → automatic authority
```

That is why the architecture uses an authority boundary instead of trying to make the agent incapable of useful work.

---

# 23. What Conductor Should Expose to an Agent

The agent-facing interface should remain a coordination interface.

Conceptually, it should support operations such as:

```text
discover work
get project/task context
claim task
report progress/activity
report blocker
resolve blocker where permitted
submit task
participate in review
accept/reject through the appropriate workflow
release abandoned work
inspect governance projection
```

The interface should not become:

```text
approve consequence
execute consequence
mutate Solvent
change authority
```

The current Conductor implementation deliberately follows this boundary.

---

# 24. What Conductor Should Never Infer

Conductor should never infer:

```text
task.accepted
    → consequence authorized

governance.status == ready
    → Conductor authorized execution

activity == execution.completed
    → external execution definitely happened

agent claims proof
    → scientific claim is true

agent is assigned
    → agent has authority

agent has capability
    → agent is assigned work
```

Each inference crosses an architectural boundary.

---

# 25. What Solvent Should Never Infer

Solvent should not infer:

```text
task exists
    → action should happen

task assigned
    → authorized

task accepted
    → authorized

scientific claim is true
    → consequence permitted

agent says "I completed it"
    → consequence occurred
```

Solvent answers the narrower authority question.

---

# 26. What the External Executor Should Never Become

The executor should not become an alternate authority engine.

A safe conceptual flow is:

```text
Solvent
    → authorization decision / authorized boundary

Executor
    → performs the externally permitted effect
```

The executor may enforce its own technical constraints, safety requirements, quota, or infrastructure policy, but those do not redefine the Solvent authority model.

---

# 27. Failure Semantics

A trustworthy autonomous system must distinguish failures.

## Agent failure

```text
agent crashes
→ work may remain active
→ Conductor provides recovery
```

## Coordination failure

```text
dependency unresolved
→ Conductor does not schedule the work as eligible
```

## Governance failure

```text
Solvent unavailable
→ governance projection is unknown/unavailable
→ Conductor does not treat that as authorization
```

## Authorization denial

```text
Solvent denies
→ agent may revise/propose another action
→ no external effect occurs through the unauthorized path
```

## Execution failure

```text
Solvent authorizes
→ executor fails
→ actual outcome belongs to executor/world state
```

## Ambiguous execution

```text
executor cannot establish whether effect happened
→ ambiguity must remain ambiguity
→ Conductor activity must not fabricate success
```

The central rule is:

> A system should not claim a stronger fact than the evidence available to the system actually supports.

---

# 28. Identity and Trust

Identity has the same separation as the other concepts.

For Conductor:

```text
authentication → actor identity
```

For Solvent:

```text
authenticated/delegated actor + exact authority state
```

For execution:

```text
actual executor identity/system
```

Do not silently equate:

```text
Conductor agent identity
=
Solvent principal
=
external executor identity
```

They may be linked in a deployment, but they are different architectural concepts.

---

# 29. Conductor v1 Scope

The current Conductor is deliberately a small, single-workspace coordination system.

The v1 model does not include:

```text
multi-tenancy
project membership
RBAC
capability matching
agent swarm management
complex scheduling
general workflow engine
domain ontology
research database
execution engine
authority engine
```

The absence of these things is intentional.

This keeps Conductor focused on its lowest common denominator:

```text
Project
Task
Dependency
Assignment
Lifecycle
Activity
```

---

# 30. Why This Can Replace a Separate PM Tool

Conductor is not intended to reproduce every feature of Jira, Linear, or similar systems.

The architectural insight is that most autonomous knowledge work requires a smaller common coordination substrate:

```text
What work exists?
Who is doing it?
What depends on what?
What is its state?
What happened?
```

Those questions are generic across domains.

The domain-specific content can remain in the workload.

Therefore the intended system is:

```text
AI agent + Conductor + domain tools + Solvent when needed
```

rather than:

```text
AI agent + separate PM tool + separate research engine
+ separate authorization system
```

The goal is to make the common substrate small enough that an additional project-management product is unnecessary for the target class of autonomous workflows.

---

# 31. Architecture Review Rules for the BM-IST POC

A separate coding/research agent reviewing this POC should use the following rules.

## Rule 1 — Classify first

For every capability or piece of data, ask:

```text
Is this:
    Agency?
    Work/Coordination?
    Authority?
    Effect?
    Domain meaning?
```

Do not implement until the category is clear.

## Rule 2 — One authoritative owner

Every important fact must have exactly one authoritative owner.

## Rule 3 — No semantic leakage

BM-IST semantics must not leak into Conductor.

Solvent semantics must not leak into generic Conductor interfaces.

## Rule 4 — No shadow authority

A Conductor projection of Solvent state must never become a second authority state.

## Rule 5 — No fabricated execution evidence

An agent report or Conductor activity record is not proof of external effect.

## Rule 6 — No capability-to-authority shortcut

Agent capability does not grant permission.

## Rule 7 — No work-to-authority shortcut

Task assignment or task acceptance does not grant permission.

## Rule 8 — No authorization-to-execution shortcut

A Solvent authorization decision does not itself establish that the effect happened.

## Rule 9 — Prefer adapters to kernel growth

If domain or deployment behavior does not require a new fundamental authority primitive, keep it outside Solvent.

## Rule 10 — Prefer generic work primitives to domain abstractions

If a concept can be represented as a task, dependency, assignment, activity, or ordinary artifact, do not add a new Conductor primitive.

---

# 32. Review Questions for the Separate Coding Agent

The reviewer should answer:

### Agent boundary

- Can the agent reason and use its own tools without Conductor becoming an agent runtime?
- Can the agent propose new work?
- Can the agent report results without automatically creating authoritative truth?

### Conductor boundary

- Can Conductor correctly represent the work?
- Can it enforce its lifecycle?
- Can it correctly handle dependencies?
- Can an agent use MCP/API without bypassing the lifecycle?
- Can abandoned work be recovered?
- Does Conductor remain domain-neutral?

### Solvent boundary

- Does consequential authorization still belong entirely to Solvent?
- Can Conductor mutate authority?
- Can Conductor create a shadow authority state?
- Is governance observation clearly non-authoritative?
- Are exact target/state semantics preserved?

### Executor boundary

- Is the actual external effect produced outside Conductor?
- Is execution outcome distinguished from authorization?
- Can an activity record be mistaken for execution proof?

### BM-IST/domain boundary

- Are mathematical/scientific concepts confined to the workload?
- Can the same infrastructure support a completely unrelated workload?
- Did any BM-IST concept become a new Conductor or Solvent primitive without a generic justification?

---

# 33. What Would Justify Changing Conductor?

A new Conductor capability should only be added when the current generic work model cannot express a genuinely common coordination requirement.

Good justification:

```text
A generic coordination invariant cannot be represented with
Project / Task / Dependency / Assignment / Activity.
```

Bad justification:

```text
BM-IST happens to need this.
```

The same principle applies to Solvent even more strongly.

A Solvent kernel change requires a genuinely new durable security fact or atomic security transition that cannot be expressed outside the existing frozen kernel.

---

# 34. What Would Justify Changing Solvent?

Only something fundamental to authority.

Examples of the right category:

```text
new exact authority fact
new atomic authority transition
new durable security invariant
```

Not:

```text
new research concept
new domain ontology
new project workflow
new agent feature
new UI feature
new experiment type
```

This is the practical meaning of:

> **Grow the ecosystem, not the kernel.**

---

# 35. Recommended POC Topology

The conceptual topology for the BM-IST trust-verifier should be:

```text
                         BM-IST DOMAIN
                    meaning / semantics / truth
                              │
                              ▼
                    ┌───────────────────┐
                    │   AI AGENTS       │
                    │     AGENCY        │
                    │ reason / propose  │
                    │ investigate / act │
                    └─────────┬─────────┘
                              │
                         work state
                              │
                              ▼
                    ┌───────────────────┐
                    │    CONDUCTOR      │
                    │   COORDINATION    │
                    │ project / tasks   │
                    │ deps / assignment │
                    │ lifecycle/activity│
                    └─────────┬─────────┘
                              │
                     governance observation
                              │
                              ▼
                    ┌───────────────────┐
                    │      SOLVENT      │
                    │     AUTHORITY     │
                    │ target / intent   │
                    │ state / authorize │
                    │ revoke / claim    │
                    └─────────┬─────────┘
                              │
                              ▼
                    ┌───────────────────┐
                    │     EXECUTOR      │
                    │      EFFECT       │
                    └─────────┬─────────┘
                              │
                              ▼
                       EXTERNAL WORLD
```

The Agent and Conductor interact continuously.

The agent may also meet Solvent directly at the consequential boundary.

The diagram should therefore not be interpreted as:

```text
Agent must always call Conductor,
then Conductor must call Solvent,
then Solvent must call Executor.
```

That is not the architecture.

---

# 36. The Deep Architectural Thesis

The system is built around a simple observation:

> Increasing machine capability does not eliminate the need for coordination or authority. It makes the distinction between them more important.

An autonomous agent can potentially do enormous amounts of work.

Conductor makes that work operationally coherent.

Solvent prevents capability or work assignment from silently becoming consequential authority.

The external executor is where the world actually changes.

This produces a chain of distinctions:

```text
Can do it
    ≠
Was assigned to do it
    ≠
Was authorized to do it
    ≠
Actually did it
```

That is the core reason Conductor and Solvent should remain separate.

---

# 37. Final Architectural Baseline

For the BM-IST trust-verifier POC, use this as the standing baseline:

```text
AGENT      = AGENCY
CONDUCTOR  = COORDINATION
SOLVENT    = AUTHORITY
EXECUTOR   = EFFECT
```

with:

```text
CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION
```

and:

```text
DOMAIN MEANING
    remains outside the three infrastructure responsibilities
```

The resulting division is:

```text
Agent
    reason, propose, investigate, implement, act, produce, report

Conductor
    project, task, dependency, assignment, lifecycle, activity

Solvent
    target, exact state binding, intent, authority, authorization,
    revocation, claim

Executor
    actual external effect
```

The architecture should be considered healthy when:

1. an AI agent can perform substantial domain work without Conductor becoming an agent runtime;
2. Conductor can coordinate that work without understanding domain meaning;
3. Solvent can govern consequential actions without understanding the domain;
4. external executors can produce effects without becoming alternate authority systems;
5. no system needs to duplicate another system's authoritative state.

That is the standard against which the BM-IST POC should be reviewed.
