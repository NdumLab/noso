# Scenario: Kubernetes crash loop with database dependency failure

## Failure class

Pod-level outage where the workload is crashing and the follow-up evidence should reveal an upstream database-connectivity problem.

## Suggested trial flow

```bash
cli-helper incident-ingest --input trials/fixtures/alerts/k8s-crashloop-db.json
cli-helper troubleshoot "worker pod alert"
cli-helper incident-status --query "worker pod alert"
```

## What to evaluate

- The seeded incident target should start from the affected pod instead of rediscovering a generic workload.
- Likely causes should include both the crash loop and the dependency failure once evidence is present.
- DNS and socket follow-ups for the database endpoint should be concrete rather than generic prose.
- If a container name is present, it should persist in follow-up log suggestions.
