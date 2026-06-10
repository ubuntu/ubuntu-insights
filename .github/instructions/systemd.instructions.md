---
description: "Use when editing or verifying systemd user units and timers under insights/autostart/systemd/."
applyTo: "insights/autostart/systemd/**/*"
---

# Systemd User Units and Timers Verification

This directory contains user-level systemd units and timers used by the client for automated insights/collection workflows.

## Verification Command

Any change to files in this directory must immediately trigger verification using `systemd-analyze`:

```bash
systemd-analyze --user verify ./insights/autostart/systemd/*
```

## Conventions

- Systemd service configurations must define clear dependencies and start-up constraints.
- Timers must be verified against their triggering targets.
