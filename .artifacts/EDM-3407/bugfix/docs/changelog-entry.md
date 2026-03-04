# CHANGELOG Entry: EDM-3407

Add this under the next release's **Bug Fixes** section. Adjust the version and date when cutting the release.

```markdown
#### Bug Fixes
- Fixed device status error message showing "unknown" during the Rebooting phase when an update failed (e.g. missing hook file or rendered spec). The message now shows a specific component such as "reading hook actions from" or "missing rendered spec" when the cause is known. (EDM-3407)
```

**Semantic versioning:** Patch (bug fix, no API or behavior change beyond message content).
