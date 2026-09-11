I read the attached writeup closely. Its architecture maps **very cleanly** onto what we have now, but there is one crucial distinction:

> **The Research State Engine described in the writeup is not Conductor. It is a domain application that can run on top of Conductor and use Solvent.**

That distinction is important because otherwise we would start putting the research ontology into Conductor and violate the domain-agnostic/non-overlap rules we just established.

The writeup itself reaches essentially this conclusion: the research engine owns the epistemic ontology and research graph, while Solvent owns durable authority, exact target/snapshot, authorization, revocation, intent claim and the execution boundary. 

## The clean mapping

I would model the three systems like this:

```text
                 HUMAN / AI AGENTS
                        │
                        ▼
              ┌─────────────────────┐
              │     CONDUCTOR       │
              │ project/work        │
              │ task lifecycle      │
              │ dependencies        │
              │ agent coordination  │
              │ activity            │
              └───────┬─────────────┘
                      │
             domain application
                      │
                      ▼
              ┌─────────────────────┐
              │ RESEARCH ENGINE     │
              │                     │
              │ claims              │
              │ hypotheses          │
              │ evidence            │
              │ proofs / attacks    │
              │ debts / obligations │
              │ research graph      │
              │ domain semantics    │
              │ research versions   │
              └────────┬────────────┘
                       │
            consequential intent
                       │
                       ▼
              ┌─────────────────────┐
              │      SOLVENT        │
              │                     │
              │ authority           │
              │ authorization       │
              │ target              │
              │ authority snapshot  │
              │ intent              │
              │ revocation          │
              │ claim               │
              │ execution boundary  │
              └────────┬────────────┘
                       │
                       ▼
                 EXECUTOR / WORLD
```

This is actually stronger than the architecture in the writeup because **Conductor fills the missing project/work coordination layer**.

---

# 1. What in the writeup belongs to Conductor?

Very little of the **research ontology** itself belongs in Conductor.

Conductor should own the **work surrounding the research**, not the meaning of the research.

For example:

| Writeup concept                            | Conductor                   |
| ------------------------------------------ | --------------------------- |
| Research project                           | **Project**                 |
| Research task                              | **Task**                    |
| Agent doing investigation                  | **Agent assignment**        |
| “Investigate hypothesis X”                 | **Task**                    |
| “Run proof search”                         | **Task**                    |
| “Review result”                            | **Task / review lifecycle** |
| Dependencies between research jobs         | **TaskDependency**          |
| Agent says it found something              | **Activity**                |
| Link to paper / commit / experiment output | **Activity reference**      |
| Human review of work                       | **Task review/acceptance**  |
| Research workflow coordination             | **Conductor coordination**  |

That is precisely why Conductor's four-table model is useful.

For example, the research system might generate:

```text
Project: Phase-2 Investigation

Task 1:
  Investigate hypothesis H17

Task 2:
  Construct candidate model

Task 3:
  Run numerical test

Task 4:
  Review result

Task 5:
  Perform expensive governed computation
```

Conductor doesn't need to understand what H17 means.

It only needs to know:

```text
Task 1 depends on Task 2
Task 3 is assigned to Agent A
Task 4 is awaiting review
Task 5 has governance_ref
```

That is exactly the domain-agnostic property we wanted.

---

# 2. What belongs to the Research Engine?

Almost everything in sections 2–13 of the writeup's research ontology belongs here.

The writeup explicitly describes the research engine as owning:

> epistemic ontology, dependency graph, proofs, attacks, semantic meaning, spectral classifications, survivor computation and research frontier. 

So these stay **outside Conductor and outside Solvent**:

```text
AXIOM
DEFINITION
HYPOTHESIS
CLAIM
THEOREM
PROOF
COUNTEREXAMPLE
ATTACK
SEMANTIC REVIEW
RESEARCH GRAPH
SPECTRAL CLASSIFICATION
SURVIVOR / FRONTIER
DOMAIN-SPECIFIC STATUS
```

The writeup is correct that these are domain semantics rather than authority primitives. 

This is particularly important for our new architecture.

We do **not** want:

```text
Conductor
  └── belief
      └── hypothesis
          └── proof
```

Nor:

```text
Solvent
  └── theorem
      └── counterexample
```

Instead:

```text
Research Application
  └── research ontology
```

---

# 3. What belongs in Solvent?

The strongest portion of the writeup is its identification of the narrow authority boundary.

The document says the research engine determines:

> what should be investigated,

while Solvent determines:

> whether this exact consequential action is authorized. 

That maps directly to our frozen Solvent architecture.

So Solvent owns:

```text
Target
Snapshot
Intent
Authority
Authorization
Revocation
Claim
Execution boundary
```

The writeup's own summary is essentially correct here. 

The critical point is that **the domain-specific meaning of the action stays outside Solvent**.

Solvent doesn't need to know:

```text
COMPUTE_KOOPMAN_SPECTRUM
PYV
p-adic
hyperbolic
Fisher rigidity
BM-IST
```

It needs the exact identity of the target/state/action and whether that consequence is authorized.

---

# 4. The subtle part: Belief / Evidence / Debt

This is where I would **modify the interpretation of the writeup** for our Conductor architecture.

The writeup says these can map onto Solvent:

```text
Axiom       → Belief
Hypothesis  → Belief
Proof       → Evidence
Attack      → Evidence
Obligation  → Debt
```

That is conceptually useful. 

But we should **not read that as “the Research Engine should store its entire research graph in Solvent.”**

That would violate our responsibility non-overlap principle.

Instead:

```text
Research Engine
  owns:
    research beliefs
    research evidence
    research obligations
    research graph
    research semantics
```

while Solvent may hold:

```text
minimal governance-relevant beliefs/evidence/debt
```

when a domain application deliberately chooses to use Solvent's primitives for governance.

So the mapping is **semantic correspondence**, not necessarily database ownership.

This is an important distinction.

---

# 5. Research Snapshot vs Solvent Snapshot

The writeup has a very interesting idea:

> `Research Snapshot S_n` = the complete accepted research state at a point in time. 

I like the idea tremendously.

But I would **not make Research Snapshot = Solvent Snapshot**.

They are different things.

### Research snapshot

Owned by the Research Engine:

```text
ResearchSnapshot
  =
  beliefs
  evidence
  debts
  dependencies
  versions
  research statuses
```

### Solvent snapshot

Owned by Solvent:

```text
AuthoritySnapshot
  =
  exact state against which authority was created
```

They can be **linked**:

```text
ResearchSnapshot RS-042
        │
        │ informs
        ▼
Solvent target/authority snapshot S-17
```

but they remain different authorities.

This is another case where the writeup contains a powerful conceptual analogy that should not become database coupling.

---

# 6. The Intent mapping is excellent

This part maps almost perfectly:

```text
Research Engine
    ↓
"Run experiment X"
    ↓
Intent
    ↓
Solvent
```

The writeup's example:

```yaml
intent:
  target: Phase2ToyModel/run-0042
  action: COMPUTE_KOOPMAN_SPECTRUM
  snapshot: research-state-042
  proposed_by: agent-koopman-03
```

is exactly the right **conceptual pattern**, though we should treat the research-specific fields as domain application data rather than automatically making them Solvent kernel fields. 

The deeper relationship is:

```text
agent proposes
      ≠
agent authorized
```

which the writeup captures very well. 

---

# 7. Conductor adds something the writeup doesn't fully have

This is the most interesting implication.

The writeup has:

```text
Research Engine
      ↓
Solvent
      ↓
Executor
```



Our architecture adds:

```text
Research Application
        ↕
    Conductor
        ↓
     Solvent
```

So now:

```text
                  CONDUCTOR
             manages the work
                    │
                    ▼
             RESEARCH ENGINE
             manages meaning
                    │
                    ▼
                SOLVENT
             manages permission
                    │
                    ▼
                EXECUTOR
             performs action
```

This gives us **four distinct responsibilities**:

### Conductor

**What work are we doing? Who is doing it? What depends on what? What happened?**

### Research Engine

**What do we believe? Why? What remains unresolved? What does this domain mean?**

### Solvent

**Is this exact consequential action authorized against the relevant state?**

### Executor

**What actually happened in the external world?**

That is a substantially cleaner architecture.

---

# 8. The writeup's final philosophical mapping fits perfectly

The attached document ends with:

> Research Engine: “What do we currently believe, why, and what remains unresolved?”
> Solvent: “Is this exact consequential action authorized against this exact state?”
> Executor: “Did the authorized action actually happen, and what was the result?” 

I would add **Conductor** in front of that:

> **Conductor:** “What work needs to happen, who is doing it, what depends on it, and what happened?”

So we get:

```text
┌─────────────┐
│  CONDUCTOR  │
│ Work state  │
└──────┬──────┘
       │
       ▼
┌────────────────┐
│ RESEARCH/DOMAIN│
│ Meaning state  │
└──────┬─────────┘
       │
       ▼
┌─────────────┐
│   SOLVENT   │
│ Authority   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  EXECUTOR   │
│ World state │
└─────────────┘
```

And the resulting four questions are beautifully orthogonal:

**Conductor:** What work is happening?
**Domain application:** What does it mean?
**Solvent:** Is the consequence authorized?
**Executor:** What actually happened?

---

## One correction I would make to the writeup

The line:

> “Research State Engine → Solvent → Executor”

is slightly too compressed for our architecture. 

I'd revise the conceptual architecture to:

```text
Agents
   │
   ▼
Conductor
   │
   ├──────────────► Domain Application
   │                    │
   │                    │ consequential intent
   │                    ▼
   │                 Solvent
   │                    │
   │                    ▼
   │                 Executor
   │                    │
   └◄── activity ───────┘
```

The **Domain Application** is where the attached research engine belongs.

That preserves our hard rule:

> **Conductor must remain domain-agnostic.**

and the equally important rule:

> **Solvent must remain a governance substrate, not a research engine.**

The attached writeup therefore isn't something we need to build *into* Conductor. It is better understood as an excellent candidate for the **first domain application that Conductor coordinates and Solvent governs**. The document itself ultimately reaches the same conclusion: Solvent does not need a “research mode”; different domain engines can become Solvent clients while the authority substrate remains unchanged. 

That is actually a very strong validation of the architecture we've arrived at.

