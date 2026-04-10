# AGENTS.md

## Project purpose
Build a plain-English CLI assistant for Linux and DevOps workflows.

## Project workflow
- Read this file before starting work.
- Also read `work-progress.txt` before starting new work so the latest progress and unfinished items are not missed.
- Treat README.txt as the long-form product and planning document.
- Prefer small, testable changes.
- After any code change, build, test, and validate before declaring completion.
- If a test fails, diagnose and fix before moving on.
- Keep code modular and production-oriented.
- Continuously append implementation progress to `work-progress.txt`.

## Current scope
- Target platform for now: RHEL 9
- Initial domains: Linux, Git, networking, systemd, package management, text processing, archives/transfer
- Future domains: Docker, containerd, Kubernetes, Terraform, Ansible, cloud CLIs

## Delivery rules
- Keep roadmap phase names as Phase1, Phase2, Phase3, not Month 1, Month 2, Month 3
- Update documentation when behavior changes
- Do not add risky auto-execution flows unless explicitly requested
- Prefer local verification and system-native command evidence where possible

## Validation rules
- Build after code changes
- Run available tests after code changes
- Validate CLI help output and basic user flows
- Record assumptions in documentation when needed

## Progress logging rules
- Record meaningful progress in `work-progress.txt` as work proceeds.
- Append progress entries after each completed task, fix, test cycle, or validation step.
- Keep entries brief but specific.
- Include what was changed, what was tested, and the result.
- If a build or test fails, log the failure and the next intended action.
- Do not overwrite previous history unless explicitly asked; always append.
- keep work-progress.txt file clean and well labled
