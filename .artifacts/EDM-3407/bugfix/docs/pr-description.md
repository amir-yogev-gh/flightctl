# Pull Request Description: EDM-3407

## Title

**[EDM-3407]: show concrete component in rebooting-phase status error instead of unknown**

---

## Summary

When a device update failed during the Rebooting phase (e.g. BeforeRebooting hook read failure or CreateRollback spec read failure), the device status showed:

`[timestamp] While Rebooting: **unknown** failed: required resource not found`

The "unknown" component was unhelpful. Root cause: the error chain from `afterUpdate` is wrapped only with `ErrPhaseActivatingConfig` and the underlying error (e.g. `ErrReadingHookActionsFrom`) is a single-wrap, so `FormatError`'s `splitWrapped` could not extract a component and fell back to "unknown".

## Fix

- **Infer component when nil:** In `FormatError`, when `splitWrapped(rest)` leaves component nil, we call a new helper `InferComponentDisplay(rest)` that walks the error chain with `errors.Is` and maps known sentinels to display strings.
- **Sentinels supported:** `ErrReadingHookActionsFrom` → "reading hook actions from", `ErrMissingRenderedSpec` → "missing rendered spec".
- **Display-only:** The inferred value is set on a small `componentDisplayError` type used only for `Message()`; no control flow or `errors.Is` checks depend on it.

## Testing

- New unit test `TestMessage/Rebooting_phase_with_inferred_component_(EDM-3407)` asserts the message contains "While Rebooting", "reading hook actions from failed", and "required resource not found".
- New `TestInferComponentDisplay` covers nil, both sentinels, and unknown error.
- **Regression:** With the inference block removed, the EDM-3407 test fails (message shows "unknown"); with the fix it passes.
- `make unit-test` and `make lint` pass.

## Before / After

| Scenario | Before | After |
|----------|--------|--------|
| Rebooting phase, hook file read failure | `While Rebooting: unknown failed: required resource not found` | `While Rebooting: reading hook actions from failed: required resource not found` |
| Rebooting phase, missing rendered spec | `While Rebooting: unknown failed: required resource not found` | `While Rebooting: missing rendered spec failed: required resource not found` |

## Manual Testing (optional for reviewers)

1. Deploy an agent with this change.
2. Configure an OS update and a BeforeRebooting hook that reads from a file; remove that file or make it unreadable.
3. Trigger the update and confirm the device condition message shows "reading hook actions from" (or the appropriate component) instead of "unknown".
