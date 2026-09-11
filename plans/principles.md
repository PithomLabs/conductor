Yes — that changes the objective substantially.

The attached writeup is useful as a **stress test**, but we should not turn its research ontology into a fourth system or even into a “Research Engine.” The real question is:

> **What is the smallest irreducible responsibility of the AI agent, of Conductor, and of Solvent, such that the three together cover autonomous software work without overlapping?**

The cleanest answer is that they are **three different kinds of control**.

## The three lowest common denominators

|                      | AI Agent                                                     | Conductor                                                   | Solvent                                                                          |
| -------------------- | ------------------------------------------------------------ | ----------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Fundamental role     | **Do**                                                       | **Coordinate**                                              | **Authorize**                                                                    |
| Fundamental question | “What should I do, and how do I do it?”                      | “What work exists, who is doing it, and what is its state?” | “May this exact consequential action happen?”                                    |
| Owns                 | Reasoning, proposed actions, produced work, reported results | Projects, tasks, dependencies, assignment, work history     | Authority, authorization, exact target/state binding, revocation, execution gate |
| Primary object       | **Action / output**                                          | **Work item**                                               | **Consequence**                                                                  |
| State                | Cognitive/operational                                        | Work state                                                  | Authority state                                                                  |
| Truth it controls    | What the agent proposes/produces/reports                     | What work is officially scheduled/in progress/reviewed      | What is actually authorized                                                      |
| Does not own         | Project lifecycle, authority                                 | Domain truth, authority                                     | Project management, agent workflow                                               |
| Failure that matters | Bad reasoning / bad work                                     | Lost coordination / wrong assignment                        | Unauthorized consequence                                                         |

That is the lowest useful decomposition.

---

# 1. AI agent: **Do**

The agent is the **active intelligence**.

Its irreducible job is:

```text
observe context
   ↓
reason
   ↓
propose action
   ↓
perform work
   ↓
produce result
```

The agent can generate:

* plans
* code
* hypotheses
* evidence
* analyses
* tool calls
* proposed consequential actions
* reported outcomes

But these are **agent outputs**, not automatically authoritative facts.

This is exactly where several concepts in the attached writeup belong conceptually:

```text
axiom
hypothesis
claim
proof
attack
analysis
result
```

These aren't necessarily separate infrastructure primitives. They are things an agent/domain workflow may produce.

The critical boundary is:

> **The agent can propose and perform; it cannot grant itself authority.**

The writeup captures this nicely as “agent proposes ≠ agent is authorized.” 

---

# 2. Conductor: **Coordinate**

Conductor should be much smaller than the writeup's research system.

Its irreducible job is:

```text
work exists
   ↓
work assigned
   ↓
work progressing
   ↓
work blocked / reviewed
   ↓
work accepted
```

So Conductor's atomic concepts are probably just:

```text
Project
Task
Dependency
Assignment
Activity
```

That is essentially what we've already arrived at.

Conductor does **not** need to understand whether a task concerns mathematics, code, security, physics, finance, etc.

It also doesn't need to understand whether the work produced:

```text
proof
theorem
simulation
feature
experiment
report
model
```

Those are agent/domain outputs.

Conductor simply says:

```text
Task 47
assigned to Agent A
status = ACTIVE
depends on Task 31
```

That is enough.

So:

> **Conductor is the work control plane, not the reasoning engine.**

---

# 3. Solvent: **Authorize**

Solvent is narrower still.

Its irreducible question is:

> **May this exact consequential action happen against this exact state?**

That is the reason the existing Solvent primitives are so powerful.

The attached writeup correctly identifies the important boundary:

```text
target
snapshot
intent
authority
revocation
claim
execution
```

and, critically:

```text
AUTHORIZE ≠ EXECUTE
```



Solvent doesn't need to understand what the agent's work *means*.

It needs to establish:

```text
WHO
WHAT
AGAINST WHAT STATE
UNDER WHAT AUTHORITY
```

and prevent the consequential action when that authority does not exist.

---

# The resulting three-way architecture

I think this is the more fundamental model you're looking for:

```text
                    AI AGENT
                 ┌─────────────┐
                 │   REASON    │
                 │   PROPOSE   │
                 │    DO       │
                 │   REPORT    │
                 └──────┬──────┘
                        │
                work/context
                        │
                        ▼
                 ┌─────────────┐
                 │  CONDUCTOR  │
                 │             │
                 │ COORDINATE  │
                 │ ASSIGN      │
                 │ TRACK       │
                 │ REVIEW      │
                 └──────┬──────┘
                        │
                governed action
                        │
                        ▼
                 ┌─────────────┐
                 │   SOLVENT   │
                 │             │
                 │ AUTHORIZE   │
                 │ BIND        │
                 │ REVOKE      │
                 │ GATE        │
                 └──────┬──────┘
                        │
                        ▼
                    CONSEQUENCE
```

But the arrows are not simply sequential.

The agent continuously interacts with Conductor:

```text
Agent ⇄ Conductor
```

while Solvent sits at the **consequence boundary**:

```text
Agent → Solvent → consequential system
```

And Conductor can **observe** Solvent:

```text
Conductor → Solvent
          read-only
```

without owning any of Solvent's truth.

---

# This also clarifies the attached writeup

The writeup currently contains several layers that are actually **not independent system layers**.

For example:

```text
belief
evidence
debt
proof
attack
hypothesis
claim
```

These are mostly **things the agent/domain workflow reasons about or produces**.

They don't need to become another infrastructure system.

Likewise:

```text
snapshot
target
intent
authority
revocation
claim
```

are fundamentally **Solvent concepts** when they concern authorization.

So I would compress the entire writeup into:

```text
AI AGENT
  reasons over domain information
  produces claims/evidence/results
  proposes actions
  performs ordinary work

CONDUCTOR
  turns work into coordinated tasks
  assigns agents
  tracks dependencies
  records work state

SOLVENT
  evaluates exact consequential actions
  binds them to exact authorized state
  revokes/denies authority
  enforces the consequence boundary
```

That is far more powerful than building a “universal research engine.”

---

# The really important distinction

There are actually **three kinds of state**:

```text
AI Agent
    ↓
Cognitive / operational state

Conductor
    ↓
Work state

Solvent
    ↓
Authority state
```

And then external systems have:

```text
World state
```

So the full chain is:

```text
Cognition
    ↓
Work
    ↓
Authority
    ↓
World
```

Or in plain language:

> **The agent thinks and acts. Conductor organizes the work. Solvent controls consequential permission. The external world records what actually happened.**

That is probably the **lowest common denominator** we should be designing around.

---

## One subtle point I'd lock in

I would **not** make “AI agent” itself a huge architectural abstraction.

The minimum agent interface may be as small as:

```text
receive work
produce proposals/results
request context
report progress
request/perform governed action
```

The agent could be Bob today, another coding agent tomorrow, or a completely different autonomous system later.

Likewise, Conductor shouldn't become “AI orchestration infrastructure,” and Solvent shouldn't become “AI safety infrastructure.”

They each solve a more fundamental problem:

```text
AI Agent   = agency
Conductor  = coordination
Solvent    = authority
```

That gives us a very clean division:

> **Agency → Coordination → Authority**

And that, I think, is the abstraction we should use to evaluate the Conductor architecture from this point forward.

