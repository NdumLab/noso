# Operator Trial Pack

This directory contains replayable operator-trial assets for the production workflow.

## Layout

- `scenarios/` -> short scenario briefs with intended failure class and evaluation focus
- `fixtures/alerts/` -> sample alert payloads for `incident-ingest`
- `feedback-template.md` -> session capture form for trial notes

## Quick start

Start a timestamped operator session:

```bash
scripts/start-operator-trial-session.sh service-missing sre-a
scripts/start-operator-trial-session.sh k8s-crashloop-db sre-b night-shift
```

Then initialize or print the scenario flow:

```bash
scripts/run-operator-trial.sh service-missing
scripts/run-operator-trial.sh k8s-crashloop-db
scripts/run-operator-trial.sh k8s-pending-taint
```

The session helper creates:

- a timestamped directory under `trials/sessions/`
- isolated audit, incident, and troubleshoot state paths
- a `session.env` file you can `source`
- a copied `feedback.md` note seeded from the template

The scenario helper then prints the exact commands to run so the trial does not reuse your normal incident or troubleshoot state.

## Trial scenarios

- `service-missing.md`
- `k8s-crashloop-db.md`
- `k8s-pending-taint.md`

## Feedback loop

After each session:

1. Copy the notes template from `feedback-template.md`.
2. Record the exact command outputs that made you distrust or accept the guidance.
3. Convert repeated failure patterns into deterministic tests before release.
