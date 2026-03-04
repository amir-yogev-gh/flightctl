# Root Cause Analysis: EDM-3407

**Issue:** [EDM-3407](https://issues.redhat.com/browse/EDM-3407) – Rollback failed before device reboot for unknown error

**Summary:** Device status shows `OutOfDate` and rollback fails with:
`[timestamp] While Rebooting: unknown failed: required resource not found`

---

## Root Cause Summary

The failure occurs during the **Rebooting** phase of a device update (phase = ActivatingConfig), when the agent is about to reboot into a new OS image. The underlying error is a **NotFound** (gRPC `codes.NotFound`), which is mapped to the user-facing message **"required resource not found"**. The **"unknown"** component appears because the error chain is not recognized as a multi-part joined error by `FormatError`, so the specific component (e.g. hook or path) is not extracted and is displayed as "unknown".

**Most likely cause:** The **BeforeRebooting** lifecycle hook runs and tries to **read hook action files** from the hook drop-in directories. If one of those files cannot be read (e.g. missing file, or path that no longer exists between glob and read), the hook manager returns `ErrReadingHookActionsFrom` (mapped to `codes.NotFound`), which is then wrapped only with `ErrPhaseActivatingConfig` in `device.go`. That produces the exact message pattern: "While Rebooting: unknown failed: required resource not found".

---

## Evidence

### 1. Message format and phase

- The user-facing message is produced in `internal/agent/device/errors/structured.go`:
  - **Phase** comes from the first error in the chain; "Rebooting" is the display name for `ErrPhaseActivatingConfig` (see `phaseDisplayNames` and `internal/agent/device/errors/errors.go`: `ErrPhaseActivatingConfig = errors.New("after update")`).
  - **Component** is the second error in a joined error (`splitWrapped`); if there is no such second part, it is shown as **"unknown"** (lines 79–81).
  - **Status message** "required resource not found" is the text for `codes.NotFound` (line 151).

So the failure happens in a code path that wraps the underlying error with `ErrPhaseActivatingConfig` and does not expose a second-level "component" in the formatted message.

### 2. Where ErrPhaseActivatingConfig is set

- In `internal/agent/device/device.go` (lines 136–137), **only** `afterUpdate()` is wrapped with `ErrPhaseActivatingConfig`:

```go
if err := a.afterUpdate(ctx, current.Spec, desired.Spec); err != nil {
    return fmt.Errorf("%w: %w", errors.ErrPhaseActivatingConfig, err)
}
```

So the error occurs somewhere inside **afterUpdate** (activation / post-sync steps), which includes the OS reboot path.

### 3. Rebooting sub-path: afterUpdateOS

- When an OS update is in progress, `afterUpdate()` calls `afterUpdateOS()` (device.go:493–498). That path does, in order:
  - `osManager.AfterUpdate`
  - **`hookManager.OnBeforeRebooting(ctx)`**
  - `specManager.CreateRollback(ctx)`
  - Status updates (Rebooting)
  - `osManager.Reboot(ctx, desired)`

Any error from these steps is wrapped only with `ErrPhaseActivatingConfig` (single `%w`), so `FormatError` sees a two-level chain (phase + one rest error). If the rest error is not a joined error, **component** stays "unknown".

### 4. NotFound in this flow

- `ErrReadingHookActionsFrom` is explicitly mapped to `codes.NotFound` in `internal/agent/device/errors/errors.go` (line 361).
- It is returned from `internal/agent/device/hook/manager.go` when reading a hook action file fails (e.g. `os.ReadFile` returns not exist):

```123:125:internal/agent/device/hook/manager.go
		if err != nil {
			return fmt.Errorf("%w %w: %w", errors.ErrReadingHookActionsFrom, errors.WithElement(f), err)
		}
```

- Hook directories involved for BeforeRebooting:
  - `/usr/lib/flightctl/hooks.d/beforerebooting/*.yaml`
  - `/etc/flightctl/hooks.d/beforerebooting/*.yaml`

If a file from the glob is missing at read time (race, or reader abstraction returning a path that does not exist), or the drop-in directory is missing and the implementation treats that as a read failure, the agent returns this NotFound and then rollback is attempted. The **reported** error is the one from the **first** failed sync (the update that never completed), not necessarily from the rollback sync itself; the status stays OutOfDate and shows "rollback failed" with this message.

### 5. Why "unknown" appears

- `FormatError` uses `splitWrapped`, which only treats errors that implement `Unwrap() []error` as having a "first" and "rest". The device layer wraps with `fmt.Errorf("%w: %w", ErrPhaseActivatingConfig, err)`. So we get phase = Rebooting and rest = hook error. If the hook error is a simple chain (e.g. single wrap) rather than a joined error with a clear second sentinel, `splitWrapped(rest)` yields `(nil, rest)`, so **component** is nil and is displayed as **"unknown"**.

---

## Timeline / Introduction

- The bug is in existing behavior: the Rebooting phase has always wrapped errors from `afterUpdate` with `ErrPhaseActivatingConfig` only, and the hook manager has always returned `ErrReadingHookActionsFrom` (NotFound) when a hook file cannot be read. The combination leads to an unclear "unknown" component and a generic "required resource not found" message.

---

## Affected Components

| Location | Role |
|----------|------|
| `internal/agent/device/device.go:136–137` | Wraps `afterUpdate` error with `ErrPhaseActivatingConfig` only (no component in message). |
| `internal/agent/device/device.go:518–563` | `afterUpdateOS`: calls OnBeforeRebooting, CreateRollback, Reboot; any error surfaces as "Rebooting" + "unknown". |
| `internal/agent/device/hook/manager.go:116–125` | `loadActions`: returns `ErrReadingHookActionsFrom` (NotFound) when a hook file cannot be read. |
| `internal/agent/device/errors/structured.go:76–101` | `Message()`: shows "unknown" when `se.Component` is nil. |
| `internal/agent/device/errors/errors.go:361` | `ErrReadingHookActionsFrom` → `codes.NotFound` → "required resource not found". |

---

## Impact Assessment

- **Severity:** **Medium** – Update fails and rollback is reported as failed; device stays OutOfDate. The actual rollback may or may not have partially succeeded; the user sees a confusing message.
- **User impact:** Users who use OS image updates (and optionally BeforeRebooting hooks) and hit a missing or unreadable hook file (or similar NotFound in this path) see "unknown failed: required resource not found" and cannot tell what resource or step failed.
- **Blast radius:** Any flow that goes through `afterUpdateOS` and then hits a NotFound (primarily BeforeRebooting hook file read failure).

---

## Hypotheses Tested

| Hypothesis | Result |
|------------|--------|
| Error comes from Rebooting phase (ActivatingConfig) | **Confirmed** – only `afterUpdate` is wrapped with `ErrPhaseActivatingConfig`. |
| NotFound is from hook manager reading hook actions | **Confirmed** – `ErrReadingHookActionsFrom` is the only NotFound in this path that fits "BeforeRebooting" and the message format. |
| "unknown" is due to missing component in error chain | **Confirmed** – `FormatError` shows "unknown" when the second-level error is not present or not a joined error. |
| Error from specManager.CreateRollback (e.g. Read(Current)) | **Unlikely** – Current is only updated on successful sync; it should exist before reboot. CreateRollback does not map to NotFound in the same way. |
| Error from osClient.Status / bootc in CreateRollback | **Unlikely** – OS status errors are mapped to `ErrGettingBootcStatus` (Internal), not NotFound. |

---

## Recommended Fix Approach

1. **Improve the error message (high priority)**  
   - Ensure the hook manager’s error is wrapped so that `FormatError` can extract a **component** (e.g. use a joined error that includes `ErrReadingHookActionsFrom` as a recognizable sentinel in the chain).  
   - Then the message can show e.g. "While Rebooting: reading hook actions from failed [for \<path\>]: required resource not found" instead of "unknown failed".

2. **Make the component robust in structured.go**  
   - When the phase is Rebooting and the status code is NotFound, consider walking the full error chain (e.g. with `errors.As` or by checking for `ErrReadingHookActionsFrom`) to set a fallback component string (e.g. "reading hook actions from") so that "unknown" is not shown when the cause is hook-related.

3. **Harden hook loading (optional but recommended)**  
   - In `loadAndMergeActions` / `loadActions`: if the glob returns no files for a directory, treat as "no hooks" (success) rather than failure.  
   - If a single file in the list fails to read (e.g. not exist), either: skip that file with a warning and continue, or return an error that includes the path (and ensure it is passed through so the component/element is visible in the status message).

4. **Documentation**  
   - Document that BeforeRebooting hooks are read from the two drop-in directories above, and that missing or unreadable files can cause the update to fail with "required resource not found" (and after fix, with a clearer message including the path or "reading hook actions from").

---

## Alternative Approaches

- **Only improve message:** Add a special case in `Message()` for phase Rebooting + NotFound to show a generic "hook or activation step" instead of "unknown" without changing the hook manager. **Pro:** Small change. **Con:** Still does not identify the exact resource (e.g. which file).
- **Skip missing hook files:** In `loadActions`, skip files that fail to read and log a warning. **Pro:** More resilient. **Con:** Can hide misconfiguration (wrong path, permissions) and change semantics (fewer hooks run than expected).
- **Retry or backoff on NotFound in afterUpdateOS:** **Not recommended** – NotFound for a missing file is not transient; retry would not help and could delay a clear failure.

---

## Similar Bugs / References

- Same structured error format is used elsewhere; any path that wraps with only one level before `FormatError` will show "unknown" when the leaf is not a joined error. Consider a pass over other `ErrPhase*` wrap sites to ensure a meaningful component is present where possible.
- EDM-3407 (this issue): [https://issues.redhat.com/browse/EDM-3407](https://issues.redhat.com/browse/EDM-3407)
- User-facing error format: `docs/user/using/managing-devices.md` (e.g. around line 1762) describes the "[timestamp] While \<Phase\>, \<Component\> failed ..." pattern.

---

## Confidence

**High (≈90%)** that:
- The failure is in the Rebooting phase (afterUpdate → afterUpdateOS).
- The underlying code is NotFound, and the most plausible source in this path is **BeforeRebooting hook** returning `ErrReadingHookActionsFrom` when a hook action file cannot be read.
- The "unknown" component is due to the error chain not exposing a second-level component to `FormatError`.

**Medium** that the exact trigger is a missing file under `/etc/flightctl/hooks.d/beforerebooting/` or `/usr/lib/flightctl/hooks.d/beforerebooting/` (or a path derived from them); the code path and error mapping are certain, but the reporter did not confirm the exact path. Improving the message (and optionally logging the path) will make future occurrences easy to confirm.
