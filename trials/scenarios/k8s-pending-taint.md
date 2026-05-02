# Scenario: Kubernetes pending pod because of taint or placement constraints

## Failure class

Scheduler-side failure where the best next step is node or scheduling inspection rather than container logs.

## Suggested trial flow

```bash
cli-helper incident-ingest --input trials/fixtures/alerts/k8s-pending-taint.json
cli-helper troubleshoot "web pod pending"
cli-helper incident-status --query "web pod pending"
```

## What to evaluate

- The planner should not prioritize pod logs for a pure scheduling blocker.
- Node-level follow-up should become concrete when a specific node name is known.
- Likely causes should move toward capacity or placement constraints rather than crash-loop or image-pull explanations.
- If only unsafe observe steps remain, `incident-observe` should refuse to run.
