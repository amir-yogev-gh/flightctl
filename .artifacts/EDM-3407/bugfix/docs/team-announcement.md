# Team Announcement: EDM-3407 Fix

## Summary

**EDM-3407** (Rollback failed before device reboot for unknown error) is fixed in branch `bugfix/EDM-3407-rollback-rebooting-unknown-error`. The fix improves the device status error message when an update fails during the Rebooting phase so that the component is no longer shown as "unknown".

## What Changed

- **Scope:** Agent only; errors package. Status message formatting when the error chain does not expose a component via the existing `splitWrapped` logic.
- **Behavior:** When `FormatError` would have shown "unknown" for the component, we now walk the error chain and match known sentinels (`ErrReadingHookActionsFrom`, `ErrMissingRenderedSpec`) and display a concrete string ("reading hook actions from", "missing rendered spec") instead.
- **Risk:** Low. No control flow, API, or integration changes. New tests added; full unit suite and lint pass.

## Severity and Urgency

**Low urgency.** This is an improvement to error message clarity, not a critical runtime fix. Normal release process is sufficient.

## Testing for QA

- Trigger an OS update that fails in the Rebooting phase (e.g. remove or corrupt a BeforeRebooting hook file under `/etc/flightctl/hooks.d/beforerebooting/` or `/usr/lib/flightctl/hooks.d/beforerebooting/`).
- Confirm device status condition message contains a specific component (e.g. "reading hook actions from") and "required resource not found", and no longer shows "unknown" for the component.

## Deployment

No special deployment steps. Agent version that includes this fix will automatically show the improved messages.

## Performance / Scaling

No impact. One additional chain walk via `errors.Is` only when component would otherwise be nil.
