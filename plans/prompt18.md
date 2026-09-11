Implement a TARGETED REMEDIATION for the two concrete gaps identified in the latest Conductor review.

This is NOT an architecture revision.

The existing Conductor architecture, AGENTS.md boundaries, Growth Gate, and four-actor model are already considered correct and should remain unchanged:

    Agent       = agency
    Conductor   = coordination
    Solvent     = authority
    Executor    = effect

Do not introduce new infrastructure responsibilities, new authority paths, new domain concepts, or new Conductor core primitives.

The remediation is strictly:

1. Make governance_ref creation explicit and unambiguous.
2. Make the temporal nature of governance observation visible in the UI.
3. Add focused tests.
4. Run the complete validation suite.

==================================================
1. GOVERNANCE REFERENCE CREATION
==================================================

Inspect the existing task creation API, MCP create-task tool, domain model, store implementation, and tests.

Confirm how `governance_ref` is currently represented and persisted.

The intended behavior is:

    create task
        |
        +-- optional governance_ref
                  |
                  v
               immutable

`governance_ref` is an opaque linkage to an external governance system.

It is NOT:
- authority
- persisted authorization state
- a Conductor policy
- something Conductor interprets
- something that can be mutated after task creation

Do NOT introduce a dedicated `set_governance` or `update_governance` operation.

Instead, make the existing creation path explicit.

Required work:

A. HTTP API

Inspect the actual `POST /v1/projects/{id}/tasks` request schema.

Ensure the request can explicitly supply an optional `governance_ref` using the repository's existing representation.

If the field already exists but is poorly documented or obscured, clarify it rather than redesigning it.

The documentation/code should make the rule obvious:

    governance_ref may be supplied when creating a task.
    Once set, it is immutable and cannot be changed or removed.

Confirm that `PATCH /v1/tasks/{id}` cannot modify it.

B. MCP

Inspect the MCP `create_task` tool and its input schema.

Ensure `governance_ref` is visibly represented there as an optional creation-time field when supported by the current implementation.

Do NOT add another governance mutation tool.

C. Persistence

Confirm the existing store/database behavior already enforces write-once semantics.

If implementation enforcement is missing, add the smallest necessary enforcement consistent with the existing architecture.

Do not introduce new domain abstractions.

D. Tests

Add or update focused tests proving:

- task creation without governance_ref works
- task creation with governance_ref works
- governance_ref is persisted correctly
- governance_ref cannot be changed after creation
- governance_ref cannot be removed after creation
- ordinary task PATCH operations do not alter governance_ref
- HTTP and MCP surfaces expose the same intended semantics

Use the repository's existing test style.

==================================================
2. GOVERNANCE OBSERVATION TIMESTAMP IN UI
==================================================

Inspect the existing governance observation flow:

    Conductor
      -> GovernanceReader
      -> external governance provider / Solvent
      -> GovernanceState
      -> HTTP/UI

Do NOT change the authority boundary.

Governance observation remains:

- read-only
- non-authoritative
- point-in-time
- external
- optional for ordinary coordination

The UI currently needs to make that temporal/non-authoritative nature visually explicit.

Required behavior:

When displaying governance state, show an observation timestamp using clear language such as:

    As of <timestamp>

or:

    Observed at <timestamp>

Prefer "As of" or "Observed at" over language that implies continuous authorization.

Example:

    Governance
    Authorized
    As of 2026-09-11 10:42 UTC
    Source: Solvent

The exact visual treatment should fit the existing UI.

Do NOT redesign the UI.

Do NOT introduce dashboards, polling systems, background refresh infrastructure, or live authorization semantics.

Do NOT imply that:

    authorized + timestamp

means authorization is currently valid.

The timestamp represents the observation, not an authority guarantee.

IMPORTANT:

First inspect the actual `GovernanceState` and provider interfaces.

If the existing backend already exposes an observation timestamp, use it.

If it does not, determine the smallest architecture-consistent place to expose one.

Prefer extending existing observation metadata rather than introducing a new subsystem.

Do not move authorization semantics into Conductor.

If adding a timestamp to the returned governance observation is necessary, keep it explicitly descriptive of the observation event, e.g.:

    observed_at

or the repository's existing equivalent.

Do not create fields such as:

    authorization_expires_at
    currently_authorized
    authority_valid_until

unless such concepts already exist in the actual repository. They would introduce semantics that do not belong in Conductor.

==================================================
3. UI HONESTY
==================================================

The final UI must not visually present governance observation as a live guarantee.

A reviewer should be able to distinguish:

    "Conductor observed this state at time T"

from:

    "Conductor guarantees this authorization is valid now"

The former is correct.

The latter is forbidden.

Preserve the existing behavior:

    Solvent unavailable
        ->
    GovernanceState.status = "unknown"
    blockers includes provider unavailable

Do not turn governance unavailability into a task failure.

Do not prevent ordinary coordination because governance observation is unavailable.

==================================================
4. DOCUMENTATION
==================================================

Make only small documentation changes necessary to remove the ambiguity.

Potential places include:

- API documentation
- MCP tool descriptions/schema
- README only if a human-facing clarification is genuinely useful
- AGENTS.md only if the existing engineering specification is actually missing the creation-time governance_ref rule

Do NOT rewrite AGENTS.md.

Do NOT duplicate architecture documentation.

The intended rule should be expressible in one concise statement:

    `governance_ref` is an optional, opaque task linkage supplied at
    creation time and immutable thereafter.

==================================================
5. TESTING
==================================================

After implementation, run the complete repository validation.

At minimum:

    go build ./...
    go test ./...
    go vet ./...

Also inspect the repository's Taskfile/Makefile/CI configuration and run the canonical lint/check commands already used by the project.

Run focused tests for:

    governance_ref creation
    governance_ref immutability
    HTTP behavior
    MCP behavior
    governance observation timestamp
    Solvent unavailable -> unknown behavior
    UI rendering of observation time

Do not weaken or delete existing tests.

Do not update tests merely to match broken behavior.

==================================================
6. REGRESSION / ARCHITECTURE CHECK
==================================================

After the changes, inspect the diff and verify:

- no new authorization path exists
- `GovernanceReader` remains read-only
- no `CheckAuthorization` method is reintroduced
- Conductor never authorizes consequences
- Conductor never executes external effects
- governance remains optional to ordinary coordination
- Solvent remains external governance/authority
- governance_ref remains opaque to Conductor
- governance_ref remains immutable after task creation
- no domain-specific concepts entered Conductor core
- no new Conductor core primitive was introduced
- no new background polling or live-authorization mechanism was introduced
- no unrelated architecture changes were made

==================================================
7. FINAL REPORT
==================================================

Do not commit or push anything.

Report:

1. What was changed for governance_ref creation clarity.
2. What was changed for governance observation timestamps.
3. Files changed.
4. Tests added/updated.
5. Full validation commands and results.
6. Confirmation that no new authorization path or architectural responsibility was introduced.
7. Final verdict:

    GREEN — targeted remediation complete

or

    RED — remaining issue(s)

The goal is a small, surgical patch that closes these two interaction-level ambiguities while leaving the already-frozen Conductor architecture intact.