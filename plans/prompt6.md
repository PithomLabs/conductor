Revision 4 is **much cleaner**, and the two issues we previously identified are genuinely resolved: the internal `GovernanceReader` is now distinct from the external governance system, and the task lifecycle no longer contradicts itself. 

However, after another adversarial pass, I found **two remaining architectural issues that I would fix before implementation**.

## 1. `accepted` is still not enforceable by Conductor

This is the most important remaining issue.

The plan defines:

> `accepted` = the task's required work **AND any required governed consequence** have both completed successfully. 

But Conductor explicitly cannot broker or perform governed execution, and `accept_task` has:

> “Governance interaction: None.” 

So Conductor has no authoritative mechanism to determine that the external consequence actually completed.

The agent could theoretically do:

```text
Agent
  → external governance system
  → execution fails
  → calls conductor_accept_task()
  → Conductor marks task accepted
```

That violates the stated meaning of `accepted`.

This is not a minor detail. It creates a mismatch between **the semantic definition of a state** and **the information available to the component that owns that state**.

### My recommendation

Do **not** make Conductor responsible for determining governed consequence completion.

Instead define `accepted` purely in terms of **Conductor-owned work acceptance**:

> `accepted` means the work represented by the task has been reviewed and accepted by the designated reviewer.

Then governed execution is an associated external activity, not part of the meaning of Conductor's terminal state.

For example:

```text
Task
  proposed
     ↓
  active
     ↓
  review
     ↓
  accepted
```

Meanwhile:

```text
External governed consequence
  requested
      ↓
  authorized
      ↓
  executing
      ↓
  succeeded / failed / ambiguous
```

Conductor can **display the external outcome reference**, but it should not pretend to own that state.

This is actually more faithful to your core philosophy:

> **A system should only make authoritative claims about facts it controls or can verify.**

---

## 2. “Conductor only knows whether `governance_ref` exists” is technically no longer true

The plan says:

> Conductor only knows whether `governance_ref` exists. 

But the same plan says:

> `provider` is a routing/implementation selector and Conductor uses it to route to the correct `GovernanceReader`. 

Therefore Conductor necessarily understands **one piece of governance_ref semantics: provider selection**.

That isn't bad. In fact, it is perfectly reasonable.

But the statement should be changed to:

> **Conductor knows whether a governance reference exists and which registered reader should handle it, but does not interpret the provider-specific reference payload or domain semantics.**

That is much more precise.

The real abstraction is:

```text
governance_ref
 ├── provider → Conductor may interpret for routing
 └── payload  → opaque
```

rather than:

```text
governance_ref → completely opaque
```

This matters because the latter is technically false and could lead a future implementation agent to make an unnecessary abstraction.

---

# One further design concern: direct Agent → external governance

I'm **not calling this a blocker**, but it deserves explicit treatment.

The architecture now says:

```text
Agent → Conductor
       project/work

Agent → External Governance System
       governed execution
```

while Conductor records:

> “reference to execution, not execution itself.” 

That means Conductor's activity history is necessarily **secondary reporting** for external consequences.

That's acceptable, and arguably desirable given your strict boundary.

But the plan should explicitly say:

> An activity record describing an external execution is an agent-reported/project-level observation, not authoritative evidence that execution actually occurred.

Otherwise someone will eventually treat:

```text
task.accepted
execution reference recorded
```

as proof of external execution.

That would undermine the separation you're trying to establish.

---

# What I would change

I would make **only these three corrections**:

### A. Change accepted semantics

From:

> required work AND required governed consequence completed

To:

> work reviewed and accepted; external governed consequence remains externally owned.

### B. Change governance_ref wording

From:

> Conductor only knows whether governance_ref exists

To:

> Conductor recognizes the provider selector for routing but treats the provider-specific payload as opaque.

### C. Clarify external execution observations

Add:

> Conductor activity concerning external execution is observational/project metadata and is never authoritative evidence of execution success.

Everything else is in very good shape.

The core separation is now quite disciplined: standalone project state, standalone SQLite, generic read-only governance boundary, and external ownership of consequential authorization/execution. 

## Final revision prompt

```text
Perform one final narrow correction pass on Conductor — Implementation Plan Revision 4.

Do NOT implement anything.
Do NOT modify Solvent.
Do NOT redesign the architecture.

There are three remaining semantic issues to correct.

1. ACCEPTED STATE OWNERSHIP

Conductor cannot authoritatively verify external governed execution because it does not broker or perform that execution.

Therefore Conductor must not define its terminal task state as requiring successful external execution.

Change accepted semantics to:

ACCEPTED = the task's work has been reviewed and accepted by the designated reviewer.

External governed consequence state remains owned by the external governance system.

Update:
- status definition
- lifecycle
- accept_task semantics
- generic workflow
- failure table
- acceptance criteria
- tests

Do not add a new Conductor state for external execution.

2. GOVERNANCE_REF SEMANTICS

The plan currently says Conductor only knows whether governance_ref exists, while also saying Conductor reads the provider field to route to a GovernanceReader.

Make this precise:

governance_ref:
  provider = routing selector understood by Conductor
  provider-specific reference payload = opaque to Conductor core

Conductor must never interpret provider-specific reference_id, metadata, domain semantics, authorization semantics, or consequence semantics.

Update the invariant and domain-agnosticity language accordingly.

3. EXTERNAL EXECUTION ACTIVITY

Clarify that activity records referring to external governed execution are project-level observations/references supplied by an actor.

They are NOT authoritative evidence that external execution succeeded.

Only the external governance system owns the authoritative execution outcome.

Update:
- Activity semantics
- failure model
- security/trust section
- end-to-end workflow

After these changes, perform one final consistency check across:
- architecture
- domain model
- schema
- lifecycle
- API/MCP
- GovernanceReader
- governance projection
- execution boundary
- failure handling
- tests
- ADR
- domain-agnosticity rules

Only declare GREEN / Implementation-Ready if all sections are internally consistent.

Do not implement.
Do not modify Solvent.
Stop after the revised plan.
```

### My verdict now

**Revision 4: YELLOW, very close.**

After those three corrections, I would give the plan the final **GREEN / implementation-approved** designation.

The resulting boundary will be particularly clean:

```text
Conductor
  = authoritative about work

External Governance System
  = authoritative about governance + consequence

Agent
  = performs the work and invokes the appropriate external systems

Domain Application
  = supplies meaning and policy
```

That is the architecture I would freeze before starting Phase 0.

