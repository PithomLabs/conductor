Back to Conductor.

**Phase 0 is cleanly complete.** The implementation agent reports all Phase 0 acceptance criteria passing: build, `go vet`, no Solvent imports, and the internal `GovernanceReader` contract exists. The repository is still only a skeleton, which is exactly where we want it at this stage. 

The important architectural boundary has survived implementation:

```text
Conductor
  ├── domain
  └── governance
        └── GovernanceReader
```

with **no Solvent dependency/import**.

### Before Phase 1, I would keep one principle front and center

Phase 1 is where the architecture can accidentally become over-engineered.

The current target should remain:

```text
Project
Task
TaskDependency
Activity
```

and nothing else.

The database should encode **Conductor's invariants**, not attempt to model agent intelligence, domain semantics, or Solvent governance.

In particular:

```text
Conductor DB
  Project
  Task
  Dependency
  Activity

        X
        │
        ├── no belief
        ├── no evidence
        ├── no debt
        ├── no authority
        ├── no authorization
        ├── no execution state
        └── no domain ontology
```

The Phase 0 output is consistent with that direction.

### One thing I would inspect carefully in Phase 1

The coding agent needs to implement **task lifecycle + persistence as one coherent invariant system**, not as CRUD with a state field.

The critical pattern is:

```text
command
   ↓
validate transition
   ↓
BEGIN TRANSACTION
   ├── mutate task
   └── append Activity
COMMIT
```

And generic task updates must not bypass the state machine.

So the real Phase 1 success criterion isn't merely:

> “CRUD works.”

It is:

> **No API or repository path can create a Conductor state that violates the domain/lifecycle invariants.**

That includes:

```text
proposed → active        only through claim
active → review          only through submit
review → accepted        only through accept
review → active          only through rejection
terminal → anything      forbidden
task + activity          atomic
claiming                  compare-and-swap
dependency                valid same-project relationship
```

This is the point where Conductor becomes a real system rather than just a schema.

I would therefore give the coding agent the **go signal for Phase 1**, with one instruction:

> **Implement only the approved Phase 1 design. Do not introduce new domain concepts, workflow abstractions, agent registries, Solvent integrations, or convenience CRUD paths that bypass the invariants.**

Phase 1 should stay boring. That is a good thing.

