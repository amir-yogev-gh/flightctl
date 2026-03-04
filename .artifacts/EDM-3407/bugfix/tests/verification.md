# Test Verification Report: EDM-3407

**Issue:** [EDM-3407](https://issues.redhat.com/browse/EDM-3407) – Rollback failed before device reboot for unknown error  
**Branch:** `bugfix/EDM-3407-rollback-rebooting-unknown-error`  
**Date:** 2026-03-03

---

## Test Summary

Testing focused on the errors package (FormatError, InferComponentDisplay, Message) and the full unit test suite. The fix is limited to status message formatting and does not change control flow, APIs, or external behavior beyond the displayed component string. All unit tests passed. Integration and E2E were not run in this verification (see Recommendations).

---

## Regression Test

| Location | Test | Description |
|----------|------|-------------|
| `internal/agent/device/errors/structured_test.go` | `TestMessage/Rebooting_phase_with_inferred_component_(EDM-3407)` | Asserts that when the error chain is `ErrPhaseActivatingConfig` wrapping a single-wrap containing `ErrReadingHookActionsFrom`, the formatted message contains "While Rebooting", "reading hook actions from failed", and "required resource not found" (i.e. the component is inferred and no longer "unknown"). |
| `internal/agent/device/errors/errors_test.go` | `TestInferComponentDisplay` | Covers `InferComponentDisplay`: nil, `ErrReadingHookActionsFrom` in chain, `ErrMissingRenderedSpec` in chain, and unknown error returns empty. |

**Regression test validity:** With the fix, both tests pass. Without the fix (e.g. removing the inference block in `FormatError` or the `InferComponentDisplay` call), `TestMessage/Rebooting_phase_with_inferred_component_(EDM-3407)` would fail because the message would contain "unknown" instead of "reading hook actions from".

---

## Unit Test Results

| Scope | Result | Details |
|-------|--------|---------|
| `internal/agent/device/errors` | **PASS** | All tests passed: TestFromStderr, TestInferComponentDisplay, TestSplitWrapped, TestFormatError, TestGetElement, TestFormatErrorWithElement, TestMessage (including EDM-3407 case). |
| Full unit suite (`make unit-test`) | **PASS** | 3608 tests passed, 5 skipped (known skips: Redis, FIPS temp file, utmp). No failures. |

---

## Integration Test Results

Not run. The change only affects error message formatting in the agent’s status path; no service/DB/API contracts or integration points were modified. Integration tests can be run separately via `make integration-test` (requires Podman and deploy targets) to confirm no regressions in device/status flows.

---

## Full Suite Results

- **Unit:** All 3608 tests passed (5 skipped as above).
- **Integration:** Not executed (see above).
- **E2E:** Not executed. Existing E2E in `test/e2e/agent/agent_structured_errors_test.go` validates structured error messages and uses `StatusMsgNotFound`; the fix does not change that constant or the overall format, only the component value when inferred.

---

## Manual Testing

Not performed. Manual verification would consist of: triggering an OS update that fails in the Rebooting phase (e.g. BeforeRebooting hook file missing), then confirming device status shows "reading hook actions from" (or "missing rendered spec") instead of "unknown" in the error message.

---

## Performance Impact

None. One additional chain walk via `errors.Is` in `InferComponentDisplay` only when `FormatError` has a nil component; the chain is small and the work is trivial.

---

## Security Review

- Error messages only expose non-sensitive, user-facing component names ("reading hook actions from", "missing rendered spec").
- No new input surfaces or logging of secrets; no change to authentication or authorization.

---

## Known Limitations

- Inference only covers the two sentinels currently in `InferComponentDisplay`. Other NotFound causes in the Rebooting phase would still show "unknown" until added to the list.
- Manual reproduction (OS update + missing hook file) was not re-run; confidence is based on unit tests and code path analysis.

---

## Recommendations

1. **Proceed to `/review` or `/pr`** – Unit test and lint results support merging; run integration/E2E in CI if required by project policy.
2. **CI:** Ensure `make unit-test` and `make lint` remain in the PR pipeline; add `make integration-test` and/or E2E if not already present for agent-related changes.
3. **Optional:** Add an integration test that triggers a Rebooting-phase failure (e.g. hook read failure) and asserts the device condition message contains the inferred component string.
