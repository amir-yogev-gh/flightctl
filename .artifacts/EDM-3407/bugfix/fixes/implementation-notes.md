# Implementation Notes: EDM-3407

**Issue:** [EDM-3407](https://issues.redhat.com/browse/EDM-3407) – Rollback failed before device reboot for unknown error

**Branch:** `bugfix/EDM-3407-rollback-rebooting-unknown-error`

---

## Summary of Changes

When a device update failed during the Rebooting phase (e.g. BeforeRebooting hook or CreateRollback), the status message showed **"While Rebooting: unknown failed: required resource not found"** because the error chain was a single-wrap (phase + leaf error), so `FormatError` could not extract a "component" and fell back to "unknown".

The fix infers a user-facing component name by walking the error chain and matching known NotFound sentinels (`ErrReadingHookActionsFrom`, `ErrMissingRenderedSpec`). When `splitWrapped` leaves component nil, we call `InferComponentDisplay(rest)` and, if non-empty, set component to a small display-only error so the message shows e.g. **"While Rebooting: reading hook actions from failed: required resource not found"** instead of "unknown".

---

## Files Modified

| File | Change |
|------|--------|
| `internal/agent/device/errors/errors.go` | Added `InferComponentDisplay(err error) string` to map known sentinels in the chain to display strings (EDM-3407). |
| `internal/agent/device/errors/structured.go` | Added `componentDisplayError` type; in `FormatError`, when component is nil, set component from `InferComponentDisplay(rest)` so Message() shows a meaningful component. |
| `internal/agent/device/errors/structured_test.go` | New TestMessage case "Rebooting phase with inferred component (EDM-3407)" to assert the message contains "reading hook actions from" and "required resource not found". |
| `internal/agent/device/errors/errors_test.go` | New `TestInferComponentDisplay` for nil, hook sentinel, missing-rendered-spec sentinel, and unknown error. |

---

## Rationale

- **Minimal change:** Only the errors package is touched; no changes to device.go, hook manager, or spec manager. Status formatting is the single place that was showing "unknown", so fixing it there covers all call paths (hook read failure, CreateRollback Read failure, etc.).
- **Extensible:** New known NotFound (or other) sentinels can be added to `InferComponentDisplay` and the same logic will surface them without further call-site changes.
- **No behavior change for existing chains:** When the error already exposes a component via `Unwrap() []error` (e.g. hook manager’s multi-%w), `splitWrapped` still extracts it and we do not override. Inference only runs when component is nil.
- **Display-only component:** `componentDisplayError` is used only for `Error()` in Message(); it is not used with `errors.Is` for control flow, so no semantic impact.

---

## Technical Debt / TODOs

- None. If more phases or components need inferred display in the future, extend the slice in `InferComponentDisplay` (and consider a table-driven approach if the list grows).

---

## Breaking Changes

None. Message format remains `[timestamp] While <Phase>: <Component> failed [for <Element>]: <Status Message>`. Only the value of `<Component>` changes from "unknown" to a concrete string when the chain contains a known sentinel.

---

## Migration Steps

None. No configuration or data migration required.
