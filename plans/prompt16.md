I reviewed `AGENTS(10).md` as the **actual Conductor engineering contract**, using the current architecture and the remediation history as the baseline. It is very close to the document I would lock, but I would make **five corrections before committing it**.

The core is excellent. It captures the essential separation, including `CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION`, explicit non-overlap, lifecycle invariants, the Solvent read-only boundary, and the growth gate.  

## 1. Fix the governance route definition

This is the most concrete issue.

The API section says:

```text
ANY /v1/tasks/{id}/governance
```

while explicitly defining governance as read-only. 

That should be:

```text
GET /v1/tasks/{id}/governance
```

The AGENTS contract should say:

> Governance observation endpoints are read-only and MUST expose only GET semantics.

Even though the handler currently behaves read-only, allowing arbitrary HTTP methods creates unnecessary ambiguity and makes future accidental mutation easier.

---

## 2. Harden the web UI rule

The current wording says:

> “Default binding: `127.0.0.1:8080` … Remote exposure requires an authenticated deployment boundary in the future.” 

That's correct for the current POC, but I'd make the normative rule stronger:

```text
The web UI MUST bind to loopback by default.

The web UI MUST NOT be exposed beyond loopback unless the deployment
provides an explicit authenticated access boundary.
```

That prevents a future coding agent from casually changing:

```text
127.0.0.1:8080
```

to:

```text
0.0.0.0:8080
```

and claiming that it is merely a configuration change.

---

## 3. Remove volatile repository facts from the durable contract

The appendix contains:

```text
Tests: 69 passing
Packages: 10
Go: 1.25.7
```



Those are useful **snapshot facts**, but they do not belong in a durable `AGENTS.md`.

The test count will immediately become stale.

I'd retain stable facts such as:

```text
SQLite
4 core tables
API / MCP / web modes
```

only when they're architectural rather than incidental.

Replace:

> `Tests: 69 passing`

with something like:

> `The full test suite MUST pass before changes are committed.`

The document already says that elsewhere. 

---

## 4. Be careful with "Conductor owns API authentication"

The document currently places:

> API authentication (API key → stable actor identity)

under “What Conductor Owns.” 

That is acceptable for the current implementation, but architecturally I'd phrase it more carefully:

> **Conductor owns the authentication boundary for access to its coordination interface.**

Why?

Because:

```text
authentication
    ≠
authorization
    ≠
Solvent authority
```

You don't want a future reader to interpret "Conductor owns authentication" as "Conductor owns the identity/authorization system."

The document correctly makes the distinction later: authentication establishes WHO is calling and does not establish per-project authorization. 

I'd make the wording consistent in both places.

---

## 5. Tighten the `governance_ref` definition

This line is good:

> “Governance reference (write-once at creation, immutable after)” 

But the document should make one more thing explicit:

> **Conductor owns the reference, not the governance represented by the reference.**

That distinction is fundamental.

I'd use:

```text
Governance reference:
    Conductor-owned opaque linkage to an external governance system.

Governance authority:
    owned exclusively by the external governance system.
```

That prevents “Conductor owns governance reference” from being misread as “Conductor owns governance.”

---

# Everything else is strong

The most important portions are already right.

### The responsibility model

```text
Agent      = Agency
Conductor  = Coordination
Solvent    = Authority
Executor   = Effect
```

is explicit and unambiguous. 

### The non-overlap rule

The document explicitly prevents Conductor from becoming:

* an authorization system
* execution broker
* research engine
* policy engine
* agent runtime
* generic PM clone

which is exactly what we need to preserve. 

### Agent trust model

This is particularly good:

> “Agents are treated as potentially buggy or adversarial callers.” 

That should absolutely remain.

### Solvent boundary

The removal of `CheckAuthorization` from `GovernanceReader` is cleaner than the prior design:

```go
type GovernanceReader interface {
    GetState(...)
}
```

and:

> “Authorization goes directly Agent → Solvent, not through Conductor.” 

I strongly approve this. It makes the separation even cleaner.

### Lifecycle invariants

The current state machine and repository-level invariants are properly documented, including the newly formalized:

```text
active → release → proposed
```

and atomic release/claim requirements. 

### Architecture growth gate

This is one of the most important sections:

> “The fact that one domain workload needs a feature is NOT sufficient.” 

That should remain essentially unchanged.

---

# One wording I would also change

This:

> “The same Conductor core must work for restaurant reservations, library systems, software engineering…” 

is useful as evidence, but I would avoid using specific application examples in the normative contract.

Better:

> **The same Conductor core MUST remain usable across unrelated domains without domain-specific core changes.**

Then mention the existing tests as evidence separately.

That makes the document less tied to today's demos.

---

# Final verdict

I'd classify this `AGENTS.md`:

**GREEN — ready to lock after minor editorial corrections.**

I would make these final changes:

```text
1. ANY /governance → GET /governance
2. Strengthen web binding rule
3. Remove volatile test/package/version counts
4. Reword authentication ownership
5. Clarify that governance_ref is a linkage, not governance authority
6. Generalize the domain-neutrality example wording
```

After those, **I would lock `AGENTS.md` and stop changing the Conductor architecture**.

The document then becomes exactly what we need: a durable guardrail against future coding agents slowly turning the small coordination substrate into a PM suite, workflow engine, agent runtime, research engine, or second authorization system.
