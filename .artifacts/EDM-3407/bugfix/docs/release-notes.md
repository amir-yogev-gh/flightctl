# Release Notes Entry: EDM-3407

## Bug Fixes

### Device agent: Clearer error message when an update fails during reboot

When a device update failed during the Rebooting phase (for example because a lifecycle hook configuration file was missing or the rollback spec could not be read), the device status showed a generic message: **"While Rebooting: unknown failed: required resource not found"**. The word "unknown" did not help identify what actually failed.

**Change:** The agent now shows a specific component in that message when it can determine the cause. For example:
- **Before:** `While Rebooting: unknown failed: required resource not found`
- **After:** `While Rebooting: reading hook actions from failed: required resource not found` (when the failure is due to reading hook action files), or `While Rebooting: missing rendered spec failed: required resource not found` (when the failure is due to a missing spec file).

**Affected versions:** All prior versions that report device update status in this format.

**Impact:** Users who saw "unknown" in Rebooting-phase errors (for example after an OS image update failed and rollback was attempted) could not tell whether the problem was hooks, spec files, or something else. With this fix, the message indicates the component that failed when it is one of the known causes above.

**Action required:** None. Upgrade to the fixed version to get the improved messages. No configuration or workflow changes are needed.
