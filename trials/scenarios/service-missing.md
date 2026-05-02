# Scenario: Missing systemd unit

## Failure class

Host-level outage question where the named workload is not actually managed by systemd and the thread should pivot away from stale service assumptions.

## Suggested trial flow

```bash
cli-helper troubleshoot "why is worker 2 not up?"
cli-helper troubleshoot "why is worker 2 not up?"
cli-helper incident-status --query "why is worker 2 not up?"
```

## What to evaluate

- The first run can start with a service-style probe.
- The second run should not get stuck repeating the same service branch forever if the unit is missing.
- Findings and likely causes should not accumulate stale duplicated prefixes.
- Incident state should reflect the current target and latest useful findings rather than stale service history.
