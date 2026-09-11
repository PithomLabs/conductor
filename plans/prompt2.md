I reviewed the attached plan adversarially. The **overall architectural direction is sound**, especially the separate repository, separate SQLite database, and strict Solvent boundary. The current document, however, is **not yet clean enough to approve for implementation**.

The biggest problem is that the plan says “domain-agnostic” while several parts quietly encode a particular domain and, more importantly, encode **Solvent-specific concepts into Conductor's core domain model**. 

## Adversarial verdict

**Status: YELLOW — revise before implementation.**

I would not reject the architecture. I would reject the **current implementation plan** because it contains several traps that will make Conductor less generic than intended.

### 1. The core domain model is not actually domain-agnostic

This is the most important issue.

The `Task` model contains:

```text
scenario_id
intent_id
```

and the plan explicitly says those are references to Solvent governance. 

That means Solvent has leaked directly into the Conductor domain model.

A genuinely domain-agnostic Conductor should not know what a `scenario`, `belief`, `intent`, `authority target`, etc. means.

The correct abstraction is something closer to:

```text
Task
   └── optional governance reference
```

where the core model knows only that:

> “This work item has an externally governed consequence.”

The Solvent adapter can then translate that generic reference into Solvent concepts.

So:

```text
BAD
Task.scenario_id
Task.intent_id

BETTER
Task.governance_ref
```

or even a generic external-reference structure.

**Solvent-specific semantics belong inside the adapter/integration layer, not `internal/domain`.**

---

### 2. The Oracle section directly violates the new requirement

The plan contains an entire Oracle-specific section and maps:

```text
Explorer → Agent
Attacker → Agent
Verifier → Agent
Claim → Task
Evidence → Activity
...
```



That is exactly the kind of domain contamination we just decided to avoid.

It also creates a subtle architectural distortion: the model starts looking like it was designed around Oracle and then generalized afterward.

I would remove Oracle entirely from this implementation plan.

Replace it with something like:

> **Domain Extension Readiness**

and test the architecture against the abstract question:

> Can an arbitrary domain-specific engineering project use Conductor without changing Conductor's core entities or lifecycle?

That is the real architectural test.

---

### 3. The “consequential action” logic is too domain-specific

The current plan says the agent determines that a task is consequential and gives examples such as:

```text
Deploy to production
Grant access
Execute destructive operation
Change security policy
Publish artifact
```



These are useful examples, but they should **not define Conductor behavior**.

Conductor cannot know what “consequential” means.

The generic model should be:

```text
ordinary work
      ↓
task reaches externally governed boundary
      ↓
governance reference/provider becomes relevant
      ↓
external governance system determines what is permitted
```

Conductor coordinates that transition; it does not classify domain consequences.

Otherwise the project layer eventually becomes another policy engine.

---

### 4. There is a contradiction in the task lifecycle

The plan declares:

```text
completed → any   (terminal state)
```

but later says that when an agent falsely reports completion:

> “Human reviews, reopens”

 

Those two statements cannot both be true.

This is not merely documentation polish. It affects the state machine.

You need to decide whether:

```text
completed
```

means genuinely terminal, or whether there is a distinction such as:

```text
completed → review → accepted
```

or:

```text
completed
↓
reopened
↓
active
```

For a minimal system, I would resist adding a large workflow.

But the plan must establish a coherent rule for **false completion, rejection, and reopening**.

---

### 5. Task claiming is underspecified and vulnerable to races

The plan says:

> “Only one agent can claim a task at a time.” 

But the schema only has:

```text
current_agent
status
```

There is no explicit compare-and-set rule in the plan.

Two agents could conceptually do:

```text
Agent A: read proposed
Agent B: read proposed

A: claim
B: claim
```

You need an atomic transaction such as:

```text
UPDATE task
SET current_agent = ?, status = 'active'
WHERE id = ?
  AND status = 'proposed'
  AND current_agent IS NULL
```

and verify exactly one row changed.

The implementation plan should specify this as an invariant, not leave it implicit.

---

### 6. `blocked_by JSON array` is a questionable shortcut

The plan deliberately keeps dependencies out of the domain model and stores them as JSON. 

This is attractive for minimalism, but it creates problems immediately:

* no foreign-key integrity
* awkward querying
* awkward concurrent updates
* difficult deletion behavior
* difficult dependency validation

There are two defensible choices:

**A. Remove dependencies from v0.**

or

**B. Add a tiny `task_dependency` table.**

I would favor **B** if dependency coordination is genuinely central to the SDLC story.

A fourth table is preferable to putting a relational concept into JSON simply to preserve an arbitrary “three tables” constraint.

---

### 7. Activity is being overloaded into an artifact system

The plan says artifacts are represented through Activity metadata. 

That is acceptable for an experiment, but we should be careful about the semantics.

An activity says:

> something happened.

An artifact says:

> this durable thing exists.

Those are not identical.

Rather than immediately adding an artifact table, I would keep the three-table model but give Activity a clear distinction between:

```text
event
reference
metadata
```

and explicitly state that Conductor does **not** own artifact storage in v0.

That prevents Activity from quietly becoming a garbage-can table.

---

### 8. Agent identity currently allows spoofing

The API example says:

```text
conductor_claim_task(task_id, agent_id)
```

and the plan treats `agent_id` as caller input. 

That is dangerous.

The authenticated identity at the Conductor boundary should establish the actor.

Conceptually:

```text
authenticated agent
        ↓
Conductor derives agent identity
        ↓
claim task
```

not:

```text
request says agent_id = bob
        ↓
Conductor trusts it
```

This mirrors the important identity/authentication distinction already established in Solvent.

The project layer can have its own agent identity system without making that identity a freely asserted request parameter.

---

### 9. Governance caching is too prescriptive

The plan hard-codes:

> “5-second TTL cache” 

That is oddly specific at architecture-plan stage.

More importantly, a governance projection is not an ordinary cache.

You should define:

```text
fresh
stale
unknown
```

semantics first.

Then choose whether caching is necessary.

I would make caching an implementation optimization rather than part of the domain contract.

Most importantly:

> **A cached governance projection must never be used as authorization.**

The plan says this philosophically, but the implementation contract should make it explicit.

---

### 10. “Agent determines governance is needed” is underspecified

The current workflow assumes the agent figures out that a task needs governance. 

That is problematic because now agent intelligence is effectively determining architecture-level policy.

A better generic mechanism is:

```text
Task
 ├── ordinary
 └── governed-reference-present
```

or an external policy/configuration tells the agent that the task crosses a governance boundary.

The agent may discover that fact, but **Conductor should not depend on an LLM's subjective classification** to maintain its system semantics.

---

### 11. The Solvent adapter is conceptually too coupled

The adapter interface contains:

```go
GetGovernanceStatus(...)
ExplainBelief(...)
```

and returns:

```text
BeliefStatus
CanPromote
CanAuthorize
```



That's a Solvent client pretending to be a generic Conductor interface.

A cleaner architecture is:

```text
Conductor
    ↓
GovernanceProvider
    ↓
Solvent implementation
```

Then the Conductor-facing contract describes generic concepts:

```text
GovernanceSnapshot
GovernanceReference
GovernanceState
GovernanceBlocker
```

The Solvent adapter translates those into beliefs, debt, authority, intents, etc.

This is probably the **single most important refactoring after removing Oracle**.

It gives you:

```text
Conductor
   ├── Solvent provider today
   ├── another governance provider later
   └── no domain knowledge
```

without weakening Solvent's role.

---

### 12. The plan's end-to-end example is too specific

The example is:

> “Implement authentication module” → “Deploy to staging” 

That's fine as a **demo scenario**, but not as the architectural basis.

The implementation plan should use something abstract like:

```text
Create project
→ decompose work
→ agent claims task
→ agent produces result
→ review
→ governed consequence requested
→ governance state consulted
→ authorized action executed externally
→ outcome recorded
→ task completed
```

Then a demonstration can instantiate that with a concrete project later.

---

# The revised verdict

I would preserve these decisions:

```text
Separate repository             ✅
Separate binary                 ✅
Standalone SQLite               ✅
No Solvent DB access            ✅
Solvent remains authority       ✅
Conductor owns project state    ✅
Minimal UI                      ✅
MCP + API                       ✅
Server-rendered UI              ✅
Read-only Solvent integration   ✅
```

I would revise these:

```text
Solvent-specific fields in Task       ❌
Oracle-specific architecture          ❌
Domain-specific consequence rules     ❌
Hard-coded governance vocabulary      ❌
JSON dependency representation        ⚠️
Agent-supplied identity               ⚠️
5-second cache as architecture        ⚠️
False-completion lifecycle            ❌
```

## Prompt I would give the coding agent

Use this as the **revision gate**, rather than asking it to rewrite the entire plan from scratch:

---

### Conductor Plan — Adversarial Revision

Review the current implementation plan and produce **Revision 2**.

The architecture remains approved in principle:

* Conductor is a standalone project layer.
* Separate repository.
* Separate binary/process.
* Standalone SQLite database.
* No direct Solvent database access.
* No Solvent kernel changes.
* Solvent remains authoritative for governance and consequential authorization.
* Conductor remains authoritative for project/work lifecycle.

However, the current plan fails the new architectural requirement:

> **Conductor must be genuinely domain-agnostic.**

Revise the plan accordingly.

### Mandatory corrections

**1. Remove all Oracle-specific architecture.**

Delete the Oracle-specific domain mapping and replace it with a generic:

`Domain Extension Readiness`

section.

Do not mention Oracle, Trust-but-Verify, physics, security verification, or any specific future application as part of Conductor's domain model.

Future applications may consume Conductor, but they must not shape its core entities.

**2. Remove Solvent-specific concepts from the Conductor core domain model.**

Do not place fields such as:

```text
scenario_id
intent_id
belief_id
authority_id
```

directly into generic Conductor entities unless there is a compelling architectural reason.

Instead design a generic external governance reference abstraction, for example:

```text
governance_ref
```

or an equivalent domain-neutral structure.

The core Conductor model must not know what a Solvent scenario, belief, authority, or intent means.

**3. Introduce a generic governance-provider boundary.**

Design:

```text
Conductor
    ↓
GovernanceProvider
    ↓
Solvent adapter
```

The Conductor-facing interface must use generic concepts.

Solvent-specific translation belongs under the adapter.

Solvent remains the first implementation of the provider.

**4. Remove domain-specific consequential-action assumptions.**

Conductor must not decide that deployment, access grants, destructive operations, etc. are intrinsically consequential.

Define only the generic concept:

```text
ordinary work
        ↓
externally governed consequence
        ↓
governance provider
```

A domain or task may indicate that external governance applies, but Conductor must not contain domain policy.

**5. Fix task lifecycle contradictions.**

Resolve the contradiction between:

```text
completed is terminal
```

and:

```text
false completion can be reopened
```

Choose and document one coherent lifecycle.

Explicitly define:

* completion
* rejection
* reopening, if supported
* cancellation
* blocked work
* invalid transitions

Do not add unnecessary states merely to solve the contradiction.

**6. Make task claiming atomic.**

Define the database-level compare-and-set behavior required so that two agents cannot simultaneously claim one task.

The plan must specify the transaction/invariant.

**7. Reconsider dependency representation.**

Critically evaluate:

```text
blocked_by JSON array
```

versus a minimal relational `task_dependency` table versus removing dependencies from v0.

Choose based on architectural integrity, not on preserving an arbitrary three-table limit.

**8. Fix agent identity semantics.**

Do not trust arbitrary `agent_id` values supplied in mutation requests.

Define how authenticated Conductor actors are mapped to project-layer agent identities.

Keep:

```text
Conductor agent identity
```

distinct from:

```text
Solvent principal
```

Do not invent identity equivalence.

**9. Rework governance caching.**

Do not make a 5-second cache an architectural requirement.

Define the semantic states:

```text
fresh
stale
unknown/unavailable
```

and explicitly state that a governance projection is never itself authorization.

Choose caching as an implementation optimization only if justified.

**10. Keep Activity semantically clean.**

Activity represents an append-only record of project events.

Do not let it silently become an artifact database.

Artifacts do not need their own table in v0, but clearly define what activity metadata can and cannot represent.

**11. Keep the end-to-end workflow domain-neutral.**

Replace examples such as:

```text
authentication module
deploy to staging
```

with a generic SDLC workflow.

Concrete demonstration scenarios can be introduced later without changing Conductor's architecture.

**12. Preserve strict responsibility boundaries.**

The final design must make this distinction explicit:

```text
Conductor:
    organizes and coordinates work

Agent:
    performs work and interacts with external systems

Governance provider:
    evaluates externally governed consequences

Solvent:
    authoritative implementation of the governance provider

Domain application:
    supplies domain semantics and policy
```

### Final domain-agnosticity test

Before finishing Revision 2, perform this test:

> Imagine Conductor is used to build a completely unrelated software system tomorrow.

The plan must demonstrate that the project can be introduced without:

* adding domain-specific tables
* adding domain-specific task states
* changing the core task model
* changing MCP fundamentals
* changing project lifecycle semantics
* embedding domain-specific policy
* modifying the Conductor core because of the application's subject matter

If any part fails that test, revise it.

### Output

Produce only the revised written implementation plan.

Do not implement anything.

Do not create files beyond the revised plan.

Do not modify Solvent.

At the end, include a concise:

**Adversarial Revision Summary**

containing:

```text
Architecture retained
Architecture changed
Domain-specific leakage removed
New invariants
Open architectural questions
Implementation readiness verdict
```

Stop after producing the document and wait for approval.

---

That is the revision I would require before giving the coding agent permission to implement. The **three biggest fixes are generic governance abstraction, removal of Oracle/domain assumptions, and making task/identity concurrency semantics precise**.

