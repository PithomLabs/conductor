Calibrated.

The attached writeup establishes the architectural position I should now use as the **standing Pithom Labs frame**, rather than treating the earlier formulations as equally live.

The core abstraction is:

$$
\boxed{\text{AGENCY}\rightarrow\text{COORDINATION}\rightarrow\text{AUTHORITY}}
$$

but these are **orthogonal responsibilities, not a strict runtime pipeline**. 

### What I should now treat as fixed

**AI / coding agent = agency**

The agent reasons, proposes, acts, produces outputs, and reports. Domain concepts such as hypotheses, proofs, attacks, analyses, etc. may be produced by the agent, but the agent does not thereby own their authoritative meaning. Domain semantics belong to the application/context in which the agent operates. 

**Conductor = coordination**

Conductor owns the work state:

$$
\boxed{\text{Project, Task, Dependency, Assignment, Activity}}
$$

It records, assigns, sequences, tracks, and reviews work. It does not become a reasoning engine, domain-ontology engine, or policy gate. 

**Solvent = authority**

Solvent's irreducible job is:

$$
\boxed{\text{May this exact consequential action happen against this exact state?}}
$$

It owns authority-related state and the consequence boundary. It does **not** execute the external effect; `AUTHORIZE ≠ EXECUTE`. 

**External executor = effect**

The actual external system performs the consequence. That remains distinct from Solvent's authorization state. 

---

### The strongest invariant to carry forward

The attached calibration elevates this above the simpler three-word slogan:

$$
\boxed{
\text{CAPABILITY}
\neq
\text{WORK}
\neq
\text{AUTHORITY}
\neq
\text{EXECUTION}
}
$$

That is the architectural test I should use whenever we're deciding whether a new feature belongs in Agent, Conductor, Solvent, an executor, or nowhere. 

This also prevents four common category errors:

$$
\text{agent capability}\not\Rightarrow\text{authority}
$$

$$
\text{task assignment}\not\Rightarrow\text{authority}
$$

$$
\text{activity record}\not\Rightarrow\text{execution fact}
$$

$$
\text{authorization}\not\Rightarrow\text{execution}.
$$

---

### The research/BM-IST work is therefore not another layer

This is an important calibration.

BM-IST, theorem proving, security research, software engineering, etc. are **domain workloads** operating through the three primitives.

So we should not create:

```text
Agent
Conductor
Solvent
Research Engine
```

as four peer infrastructure layers.

Instead:

```text
                  DOMAIN WORK
            ┌────────────────────┐
            │ BM-IST / coding /  │
            │ security / science │
            └─────────┬──────────┘
                      │
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
      AGENT        CONDUCTOR      SOLVENT
      AGENCY      COORDINATION    AUTHORITY
```

Domain meaning is carried by the workload/application; Conductor and Solvent remain domain-agnostic. The calibration explicitly says not to turn beliefs, evidence, debt, proof, attack, etc. into another infrastructure layer. 

---

### One further calibration I will apply

I should no longer describe:

$$
\text{Agent}\rightarrow\text{Conductor}\rightarrow\text{Solvent}
$$

as though every action literally traverses those three components in sequence.

The better model is:

$$
\boxed{
\text{Agency, Coordination, Authority are separate control planes.}
}
$$

An agent interacts continuously with Conductor for work state, while consequential actions may reach Solvent directly at the authorization boundary. Conductor can observe Solvent's governance state without owning that authority. 

That distinction should now be presumed in future architecture discussions.

---

## The Pithom Labs primitive

So going forward, my default question will be:

> **Is this about doing, coordinating work, authorizing consequence, or producing the consequence itself?**

Mapped as:

$$
\boxed{
\begin{array}{ccl}
\text{Agent} &\rightarrow& \textbf{DO}\\
\text{Conductor} &\rightarrow& \textbf{COORDINATE}\\
\text{Solvent} &\rightarrow& \textbf{AUTHORIZE}\\
\text{Executor} &\rightarrow& \textbf{EFFECT}
\end{array}}
$$

with the invariant:

$$
\boxed{
\text{CAPABILITY}\neq\text{WORK}\neq\text{AUTHORITY}\neq\text{EXECUTION}.
}
$$

That is the calibrated architectural baseline I will use from here. 




## chatgpt

Yes. This is **well calibrated** and is consistent with the Conductor architecture we have now. I would make only a few precision adjustments before treating it as the standing Pithom Labs frame.

### What is correct and should be locked

The central abstraction is right:

```text
AGENCY → COORDINATION → AUTHORITY
```

with the explicit caveat that these are **responsibility boundaries, not a mandatory runtime sequence**.

The stronger invariant is also exactly right:

```text
CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION
```

That is more useful architecturally than the three-word slogan because it catches category errors.

Your division is also correct:

```text
Agent      → Agency
Conductor  → Coordination
Solvent    → Authority
Executor   → Effect
```

Conductor's current model—Project, Task, Dependency, Assignment, Activity—fits that definition cleanly. The current Conductor plan explicitly makes project/work state its responsibility and keeps governance as a read-only observation. 

And the non-overlap invariant is exactly what we wanted: every authoritative fact has one owner, with Conductor owning work state and Solvent owning governance/authority. 

### One important precision: Solvent vs Executor

I would slightly change this sentence:

> **Solvent = authority ... It owns the authority-related state and the consequence boundary. It does not execute the external effect.**

That is correct **provided “Solvent” means the authority/governance layer**, not every service living around the Solvent kernel.

The sharper formulation is:

```text
Solvent
  = authority + authorization boundary + execution gate

Executor
  = performs the external effect
```

So:

```text
Solvent:  "May this happen?"
Executor: "Make it happen."
```

This preserves the important `AUTHORIZE ≠ EXECUTE` distinction while allowing the Solvent ecosystem to contain execution adapters/services without making the kernel itself an executor.

### One terminology change

I would avoid saying:

> “three primitives”

for Agent, Conductor and Solvent.

They are better described as:

> **three irreducible responsibilities**
> or
> **three control responsibilities**

because Agent/Conductor/Solvent are systems/components, whereas “primitive” has a more specific architectural meaning.

### One nuance about domain workloads

This is also right:

> BM-IST, theorem proving, security research, software engineering, etc. are domain workloads rather than another infrastructure layer.

I would make one tiny correction:

```text
Domain workload
     ↓
uses agency
     ↓
uses coordination
     ↓
optionally crosses authority boundary
```

Not every workload necessarily needs Solvent for every action. Solvent becomes relevant **when the workload reaches a consequential boundary**.

That keeps Solvent from becoming mandatory plumbing for ordinary work.

### The calibrated model I'd lock

I would use this wording from now on:

```text
                    DOMAIN WORK
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
       AGENCY        COORDINATION     AUTHORITY
       Agent          Conductor        Solvent
          │              │              │
          │              │              │
          ▼              ▼              ▼
       outputs        work state      authority state
          │                              │
          └──────────────┬───────────────┘
                         ▼
                      Executor
                         │
                         ▼
                    external effect
```

And the standing invariant:

```text
CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION
```

with:

```text
Agent      = DO
Conductor  = COORDINATE
Solvent    = AUTHORIZE
Executor   = EFFECT
```

The most important practical consequence is:

> **Do not add something to Conductor merely because an agent needs it, and do not add something to Solvent merely because an action is important. First classify the responsibility.**

That is exactly what keeps Conductor from becoming an agent framework and Solvent from becoming an application/policy engine. The Conductor plan's current responsibility boundaries already embody this separation. 

So I would mark your proposed calibration **GREEN and standing** with those wording refinements.




