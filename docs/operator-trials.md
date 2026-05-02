# Operator Trials

`noso` is in a production-hardening phase. Operator trials are how we decide whether the incident workflow is actually trustworthy under pressure rather than merely well-tested in unit cases.

This document defines a small, repeatable workflow for running outage trials and converting weak sessions into deterministic regressions.

## Goal

Evaluate the primary production workflow end to end:

`incident-ingest` -> `troubleshoot` -> `incident-observe` -> `incident-status` or `incident-history` -> `incident-resolve`

Each trial should answer:

- Did `noso` choose the right first probe?
- Did it keep the right target, namespace, and object family across turns?
- Did it avoid unsafe or irrelevant observe actions?
- Did the likely-cause ranking become more correct as evidence accumulated?
- If it was wrong, can we replay that failure as a deterministic regression?

## Trial pack

Use the replayable assets under [trials/README.md](/opt/noso/trials/README.md).

The pack includes:

- scenario briefs
- sample incident-ingest payloads
- a feedback capture template
- a helper script to create isolated local state for a trial run

## Recommended workflow

1. Pick one scenario from the trial pack.
2. Start a timestamped session with `scripts/start-operator-trial-session.sh <scenario> <operator-id>`.
3. `source` the generated `session.env` file so the trial uses isolated local state.
4. Run the scenario commands exactly as written first.
5. If `noso` goes wrong, keep the bad output and write down the correction instead of editing the state by hand.
6. Complete the generated feedback note immediately after the run while the failure mode is still fresh.
7. For every repeated or trust-breaking failure, add or extend a deterministic test before shipping.

## Capture requirements

Capture all of the following for each session:

- operator role and environment notes
- session identifier
- chosen scenario name
- exact query or ingest payload used
- first command suggested by `noso`
- whether the operator agreed with that first probe
- what the operator would have done instead
- whether stateful follow-up improved or degraded the investigation
- whether `incident-observe` stayed within safe read-only expectations
- final trust verdict: usable, noisy, misleading, or unsafe

## Converting failures into regressions

Use this mapping:

- bad first probe -> scenario or planner regression in `internal/troubleshoot`
- stale target, namespace, or object family -> thread or incident-state regression
- unsafe observe action or fallback -> `internal/incident/observe_test.go`
- misleading likely-cause ranking -> `internal/troubleshoot/state_test.go` or scenario tests
- unclear wording but correct logic -> docs or rendering fix

The default rule is simple:

If a failure would make an experienced SRE distrust the next suggestion, it needs a deterministic regression before release.
