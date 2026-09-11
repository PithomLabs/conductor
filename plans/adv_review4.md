# CONDUCTOR — SCOPED ADVERSARIAL CODE REVIEW (REMEDIATION)

**Repository:** `/home/chaschel/Documents/go/conductor`
**Review Date:** 2026-09-11
**Reviewer Role:** Senior Go Architect / Adversarial Reviewer
**Scope:** Targeted remediation for governance_ref creation/immutability, governance observation timestamps, UI presentation of observation time, MCP schema/implementation consistency, and regression safety.
**Mode:** REVIEW ONLY — no code modifications.

---

## Verdict

    AMBER — small scoped gap remains

---

## Findings

### Finding 1 — MEDIUM: `GovernanceService.GetState` omits `RefreshedAt` for unknown-provider observations

- **File:** `internal/service/governance_service.go`, function `GetState`
- **What was expected:** All point-in-time governance observations should populate `RefreshedAt` with the time Conductor obtained the observation, including the unknown-provider path.
- **What the code actually does:** When `ref.Provider` is not registered, the service returns:
  ```go
  return &governance.GovernanceState{
      Reference: ref,
      Status:    governance.GovernanceStatusUnknown,
      Blockers:  []string{fmt.Sprintf("unknown governance provider: %s", ref.Provider)},
  }, nil
  ```
  `RefreshedAt` is left at the zero value. Every other observation path sets it:
  - `internal/governance/null_reader.go`: `RefreshedAt: time.Now()`
  - `internal/adapter/solvent/translator.go`: `RefreshedAt: time.Now()`
  - `internal/api/governance_handler.go` error fallback: `RefreshedAt: time.Now()`
  - `internal/mcp/tools.go` error fallback: `RefreshedAt: time.Now()`
- **Why it matters:** The service layer is a first-class observation path. A caller traversing HTTP or MCP will receive a `GovernanceState` with `status: "unknown"` and `RefreshedAt: 0001-01-01 00:00:00 +0000 UTC`. The web UI then omits the "As of ..." line, so the user sees an unknown-state observation with no temporal anchor. This is inconsistent with every other observation path and breaks the invariant that `RefreshedAt` communicates "time at which Conductor obtained the observation."
- **Recommended smallest fix:** Add `RefreshedAt: time.Now()` to the unknown-provider return in `GovernanceService.GetState`:
  ```go
  return &governance.GovernanceState{
      Reference:   ref,
      Status:      governance.GovernanceStatusUnknown,
      Blockers:    []string{fmt.Sprintf("unknown governance provider: %s", ref.Provider)},
      RefreshedAt: time.Now(),
  }, nil
  ```

### Finding 2 — LOW: Timestamp/UI test coverage gap

- **File:** `internal/web/web_test.go`, `TestTaskViewWithGovernance`
- **What was expected:** Direct evidence that the UI renders the "As of <timestamp>" line, omits it correctly when `RefreshedAt` is zero, and does not misrepresent stale observation as current authority.
- **What the code actually does:** The existing `TestTaskViewWithGovernance` asserts only that the governance panel heading and the observational note are present. It does not assert:
  - that "As of" appears when `RefreshedAt` is non-zero,
  - that the timestamp line is absent when `RefreshedAt` is zero,
  - that provider-unavailable/error fallback states render distinguishably.
- **Why it matters:** The UI change is the highest-risk part of the remediation from a human-factors perspective. The implementation is correct, but without direct assertions on rendered timestamp text, a future template change could silently reintroduce the misleading "Authorized" presentation.
- **Recommended smallest fix:** Add a web test that exercises `handleTask` with a non-zero `RefreshedAt` and asserts the rendered body contains `As of`, and another that exercises zero `RefreshedAt` and asserts the timestamp line is absent.

No other findings. No architectural drift detected.

---

## Evidence

- **Build:** `go build ./...` — PASS (no output)
- **Tests:** `go test ./...` — PASS (all packages cached/ok)
- **Vet:** `go vet ./...` — PASS (no output)
- **Tests inspected:**
  - `internal/store/store_test.go`: 3 new governance_ref tests (`TestTaskCreateWithGovernanceRef`, `TestTaskGovernanceRefImmutabilityViaUpdate`, `TestTaskGovernanceRefImmutabilityViaLifecycle`)
  - `internal/api/api_test.go`: 3 new HTTP tests (`TestHTTPCreateTaskWithGovernanceRef`, `TestHTTPCreateTaskWithoutGovernanceRef`, `TestHTTPUpdateTaskDoesNotAlterGovernanceRef`)
  - `internal/mcp/mcp_test.go`: 3 new MCP tests (`TestMCPCreateTaskWithGovernanceRef`, `TestMCPCreateTaskWithoutGovernanceRef`, `TestMCPSchemaAdvertisesGovernanceRef`)
  - `internal/web/web_test.go`: 1 existing governance UI test (`TestTaskViewWithGovernance`)
- **governance_ref coverage:** Fully exercised across store round-trip, HTTP create/update, MCP create/update, and lifecycle immutability.
- **Timestamp/UI coverage:** Partial — the UI renders "As of ..." correctly and omits it for zero, but no test asserts the rendered timestamp text or the zero-value omission.

---

## Architecture Check

- GovernanceReader is read-only: **TRUE** (`internal/governance/reader.go` — single `GetState` method, no mutation)
- CheckAuthorization remains absent: **TRUE**
- governance_ref is creation-time and immutable: **TRUE** — enforced by `TaskUpdateFields` exclusion, lifecycle SQL never touching `governance_ref`, and no alternate mutation path exists in the codebase
- governance remains optional to ordinary coordination: **TRUE** — web UI, HTTP, and MCP only call `GetState` when `GovernanceRef != nil`; ordinary task CRUD and lifecycle proceed without governance
- Conductor remains non-authoritative: **TRUE** — `GovernanceState` is informational only; note in UI template reinforces this
- Conductor remains non-executing: **TRUE** — no execution paths added
- no new core primitive was introduced: **TRUE**
- no domain leakage was introduced: **TRUE** — `governance_ref` remains an opaque string linkage

---

## Final Recommendation

The repository is **ready to proceed after remediating Finding 1**.

The targeted remediation correctly closes the original five concerns:
1. `governance_ref` round-trips through creation, is omitted cleanly, and has no mutation path via PATCH or lifecycle.
2. MCP schema advertises `governance_ref` as an optional string with write-once semantics, and the implementation honors it.
3. HTTP create accepts and persists `governance_ref`; PATCH cannot alter it.
4. `RefreshedAt` is correctly populated on all observation paths **except** the `GovernanceService` unknown-provider path, which is a scoped defect.
5. The UI renders an explicit "As of <timestamp>" line and omits it for zero values.

The single outstanding issue is that `internal/service/governance_service.go` should set `RefreshedAt: time.Now()` on the unknown-provider branch so timestamp semantics are uniform across all observation paths. Once that is fixed, the remediation is fully closed.
