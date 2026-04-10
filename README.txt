# Plain-English CLI Assistant

## Product document and execution plan

## 1. Working product summary

A safe, context-aware terminal assistant that helps users ask questions in plain English and get:

* the right command
* an explanation of what the command does
* verified flags from the local system where possible
* troubleshooting guidance
* expected output
* risk level before anything destructive is suggested

This product is designed for Linux users, DevOps engineers, platform engineers, SREs, cloud engineers, and learners who work in terminals daily and want faster, safer, and more explainable command-line operations.

---

## 2. Problem statement

Many terminal users know what they want to do, but do not always remember:

* the exact command
* the correct flags
* distro-specific differences
* the safest troubleshooting order
* whether a command is risky

This creates a context-switching tax.

Imagine a scenario where an engineer wants to check why a service is failing. They leave the terminal, search online, compare 4 different answers, return to the terminal, and still are not sure which command is safest. That costs time and increases error risk.

The same pattern appears with:

* Linux administration
* Git recovery tasks
* Docker troubleshooting
* containerd troubleshooting
* Kubernetes debugging
* Terraform and Ansible workflows
* cloud CLI commands

The opportunity is to turn plain-English intent into trustworthy terminal guidance while staying grounded in the real system.

---

## 3. Vision

Build the most trusted plain-English terminal copilot for operators.

Not just a command generator.

A tool that:

* understands what the user wants
* checks what is actually installed on the system
* verifies commands and flags locally where possible
* explains what each command does
* helps troubleshoot step by step
* separates read-only inspection from risky fix actions

---

## 4. Product goals

### Primary goals

* Reduce time spent searching for commands
* Reduce errors caused by incorrect flags or wrong commands
* Improve troubleshooting speed
* Teach users while helping them work
* Build trust through explainability and local verification

### Secondary goals

* Support local-first and privacy-sensitive environments
* Create reusable runbooks from troubleshooting sessions
* Become useful enough to live permanently in the terminal workflow

---

## 5. Non-goals for early versions

To keep scope under control, the product will not initially:

* auto-execute destructive commands by default
* attempt full autonomous remediation without explicit user approval
* support every CLI tool on day one
* depend on internet connectivity for core value
* replace official documentation entirely

---

## 6. Target users

### Primary users

* Linux administrators
* DevOps engineers
* SREs
* platform engineers
* cloud engineers
* support engineers

### Secondary users

* junior engineers learning CLI workflows
* developers working with Git, Docker, containerd, and Kubernetes
* students and lab users
* security and operations teams

### Example user profiles

#### 1. Senior DevOps engineer

Needs fast, accurate command recall and troubleshooting support across Linux, Git, Docker, containerd, Kubernetes, Terraform, and cloud CLI.

#### 2. Junior Linux admin

Knows the task but not always the exact syntax. Benefits from explanation mode and safer suggestions.

#### 3. On-call engineer

Needs quick diagnostic paths under pressure and wants low-risk commands first.

---

## 7. Core value proposition

### Main value

Plain-English terminal help that is:

* safe
* explainable
* environment-aware
* locally verified where possible
* useful for both command discovery and troubleshooting

### Why this wins

Existing tools often do one of the following:

* generate commands without strong local validation
* provide generic chatbot answers
* do not understand the machine state
* do not classify risk
* do not help build troubleshooting paths

This product wins by being:

* Linux-first
* operator-friendly
* local-evidence grounded
* strong at troubleshooting
* strong at teaching

---

## 8. Key differentiators

### 1. Local verification first

Commands and flags should be sourced from the host when possible using:

* `command -v`
* `type`
* `help`
* `--help`
* `man`
* `whatis`
* `apropos`
* completion scripts
* package metadata
* installed docs

### 2. Explainable output

Each answer should show:

* command
* explanation
* expected result
* risk level
* confidence level
* evidence source when available

### 3. Troubleshooting mode

The tool should not only answer “what command do I run?” but also:

* suggest the first safe checks
* interpret outputs
* propose next likely steps
* build a decision path

### 4. Context awareness

The tool should detect:

* distro and package manager
* shell type
* whether it is inside a Git repo
* whether Docker, Podman, or containerd tooling is installed
* whether Kubernetes context is active
* which commands are available locally

### 5. Safe-by-default behavior

The default should be:

* inspect first
* explain second
* suggest fix third
* require explicit user approval for risky actions

---

## 9. Product principles

1. Trust over cleverness
2. Local evidence over guessed syntax
3. Read-only first
4. Teach while helping
5. Adapt to the real host
6. Keep terminal users in flow
7. Avoid magic that cannot be explained

---

## 10. Product scope

## MVP scope

### Domains

* Linux core commands
* systemd and service management
* networking
* package management
* users and permissions
* Git
* text processing tools
* archives and transfer tools

### Supported example tools

* `ls`, `cp`, `mv`, `rm`, `find`, `cat`, `tail`, `less`, `df`, `du`, `ps`, `kill`
* `systemctl`, `journalctl`
* `ss`, `ip`, `ping`, `dig`, `curl`, `wget`, `nc`
* `dnf`, `yum`, `rpm`, `apt`, `dpkg`
* `id`, `chmod`, `chown`, `sudo`, `getfacl`, `setfacl`
* `git`
* `grep`, `awk`, `sed`, `cut`, `sort`, `uniq`, `jq`, `yq`
* `tar`, `rsync`, `scp`, `ssh`

### MVP capabilities

* plain-English to command suggestions
* command explanation
* local command existence checks
* flag validation from system-native sources where possible
* troubleshooting trees for common issues
* risk classification
* expected output guidance
* shell history or audit log of interactions

### Current implemented Phase1 coverage

The current codebase supports:

* ask mode for files, processes, disk, services, networking basics, package inspection, text search, archives, Git inspection, and read-only containerd inspection
* explain mode for in-scope commands such as `git reset --hard`, `rm -rf`, `systemctl status`, `journalctl`, `ss`, `find`, `du`, `df`, `grep`, `rpm -qi`, `tar -tf`, `ip addr`, `ping`, and `curl -I`
* starter troubleshooting mode for service startup failures, basic connection issues, and disk-full scenarios
* environment reporting for RHEL 9 and available local tooling

Example queries currently supported:

* `what process is using port 8080`
* `show disk free space`
* `package info for bash`
* `show git log`
* `containerd status`
* `explain git reset --hard HEAD~1`
* `nginx is not starting`

---

## 11. Phase 2 scope

### Additions

* Docker
* containerd
* Podman
* Kubernetes
* Helm
* Terraform
* Ansible
* stronger SSH/remote workflows

### Example questions supported

* why is this container unhealthy
* why is my pod in CrashLoopBackOff
* how do I preview a Helm release
* how do I validate Terraform before apply
* how do I run an Ansible dry run

### Phase2 Milestone 1

The first Phase2 milestone focuses on container runtime foundations:

* Docker, Podman, and containerd detection
* read-only runtime inspection for status, version, containers, images, logs, and inspect output
* starter troubleshooting for unhealthy containers, failed starts, and image pull failures
* no runtime execution or remediation automation by default

### Phase2 Milestone 2

The next Phase2 milestone focuses on Kubernetes foundations:

* `kubectl` detection and current-context visibility
* read-only inspection for pods, deployments, services, namespaces, logs, describe output, and events
* starter troubleshooting for CrashLoopBackOff, ImagePullBackOff, and Pending pods
* no apply, delete, rollout restart, or other cluster-mutating commands by default

### Phase2 Milestone 3

The next Phase2 milestone focuses on Helm foundations:

* `helm` detection and client version visibility
* read-only inspection for repos, releases, release status, history, and values
* dry-run-first local template rendering for chart previews
* no install, upgrade, rollback, or uninstall guidance by default

### Phase2 Milestone 4

The next Phase2 milestone focuses on Terraform foundations:

* `terraform` detection and CLI version visibility
* local-only validation and preview flows for formatting, validation, plan output, workspaces, and state listing
* no apply, destroy, import, or state mutation guidance by default

### Phase2 Milestone 5

The next Phase2 milestone focuses on Ansible foundations:

* `ansible` and `ansible-playbook` detection plus version visibility
* inventory inspection, playbook syntax-check, and playbook check-mode preview flows
* no broad ad-hoc mutation or normal playbook execution guidance by default

### Phase2 Milestone 6

The next Phase2 milestone focuses on stronger SSH and remote workflows:

* SSH client version, host configuration inspection, and host-key review
* safe SSH port reachability checks
* rsync dry-run transfer previews and safer SCP-style preview guidance
* no unattended remote command execution or broad remote mutation guidance by default

---

## 12. Phase 3 scope

### Additions

* AWS CLI
* Azure CLI
* GCP CLI
* Argo CD
* security and compliance tooling
* hardware diagnostics
* GPU / AI infrastructure tooling
* database CLI support

### Phase3 Milestone 1

The first Phase3 milestone focuses on cloud CLI foundations:

* `aws`, `az`, and `gcloud` detection plus CLI version visibility
* read-only identity and account or project inspection for each cloud CLI
* local profile or subscription visibility where the CLI supports it safely
* no create, deploy, delete, or apply-style cloud actions by default

### Phase3 Milestone 2

The next Phase3 milestone focuses on Argo CD foundations:

* `argocd` detection plus CLI version visibility
* read-only account, application, project, and cluster inspection
* no sync, rollback, delete, or app mutation guidance by default

### Phase3 Milestone 3

The next Phase3 milestone focuses on security and compliance foundations:

* SELinux mode and status inspection
* firewalld zone and rule inspection
* OpenSSL certificate decoding and inspection
* no enforcement or firewall mutation guidance by default

### Phase3 Milestone 4

The next Phase3 milestone focuses on hardware diagnostics and GPU or AI infrastructure foundations:

* CPU, memory, block-device, and system identity inspection
* SMART disk-health inspection where `smartctl` is present
* basic IPMI controller visibility where `ipmitool` is present
* NVIDIA GPU status and utilization visibility where `nvidia-smi` is present
* no firmware changes, hardware tuning, or device mutation guidance by default

### Phase3 Milestone 5

The next Phase3 milestone focuses on database CLI foundations:

* PostgreSQL client version and database listing visibility
* MySQL client version and database listing visibility
* Redis CLI version and ping or health visibility
* no schema changes, data writes, flushes, or destructive administrative commands by default

---

## 13. Phase 4 scope

### Additions

* output interpretation mode
* product self-check and environment readiness
* packaging and installability foundations

### Phase4 Milestone 1

The first Phase4 milestone focuses on output interpretation foundations:

* an `interpret` CLI mode for pasted command output
* targeted interpreters for common operational commands such as `systemctl status`, `df -h`, `free -h`, and `kubectl get pods`
* read-only analysis that summarizes likely meaning and suggests the next safe checks
* no automatic command execution based on interpreted output

### Phase4 Milestone 2

The next Phase4 milestone focuses on product self-check foundations:

* a `doctor` mode for local readiness checks
* validation of config, writable audit path, and key runtime dependencies
* actionable warnings when the local environment limits accuracy or supported workflows

### Phase4 Milestone 3

The next Phase4 milestone focuses on packaging foundations:

* explicit CLI version reporting
* install or setup documentation for local use
* release-friendly local build, test, and install scripts
* no network-dependent packaging assumptions by default

---

## 14. Phase 5 scope

### Additions

* audit history inspection
* runbook generation from local sessions
* exportable markdown or JSON incident summaries

### Phase5 Milestone 1

The first Phase5 milestone focuses on audit history foundations:

* a `history` CLI mode for reading local audit records
* support for recent-entry listing and simple query filtering
* read-only session inspection using the existing audit log
* no remote sync or shared history backends by default

### Phase5 Milestone 2

The next Phase5 milestone focuses on runbook generation foundations:

* a `runbook` CLI mode that summarizes recent audit activity
* problem summary, commands used, findings, and next-step sections
* markdown and JSON export support from local session history
* no team sync, remote storage, or multi-user collaboration by default
### Example advanced tools

* `aws`, `az`, `gcloud`
* `argocd`
* `getenforce`, `sestatus`, `firewall-cmd`, `openssl`
* `ipmitool`, `smartctl`, `dmidecode`, `nvidia-smi`
* `psql`, `mysql`, `redis-cli`

---

## 13. User experience design

## Main interaction modes

### 1. Ask mode

User asks a plain-English question.

Example:

> how do I find files larger than 1G in /var

Response includes:

* command
* explanation of flags
* expected output
* risk level

### 2. Explain mode

User provides a command and asks what it does.

Example:

> explain `git reset --hard HEAD~1`

Response includes:

* breakdown of each part
* what changes will occur
* whether data can be lost

### 3. Troubleshoot mode

User describes a problem.

Example:

> nginx is not starting

Response includes:

* likely causes
* first safe checks
* interpretation guidance
* next steps based on results

### 4. Analyze output mode

User pastes command output.

Example:

> here is the output of `systemctl status nginx`

Response includes:

* plain-English interpretation
* likely issue
* next command to confirm the diagnosis

### 5. Runbook mode

At the end of a session, generate:

* problem summary
* commands used
* findings
* root cause hypothesis
* suggested fix
* validation steps

---

## 14. Example response format

### Example: command answer

Question:

> how do I see what process is using port 8080

Answer:

```bash
ss -ltnp | grep :8080
```

Why this command:

* `ss` shows listening sockets
* `-l` shows listening ports
* `-t` limits to TCP
* `-n` avoids DNS lookups
* `-p` shows process information

What to expect:

* a matching process and PID if something is listening on port 8080

Risk:

* Low

Confidence:

* High

Verified from:

* local `ss --help`
* command exists on system

---

## 15. Architecture overview

## High-level architecture

### 1. CLI frontend

Handles:

* user questions
* mode selection
* output formatting
* command preview
* optional execution workflow later

Possible implementation:

* Go standalone binary

### 2. Intent parser

Converts plain-English request into structured intent.

Examples:

* `find_large_files`
* `explain_git_command`
* `troubleshoot_service_failure`
* `inspect_k8s_pod`

This can be rule-based at first, with model support where needed.

### 3. Environment detector

Collects local context such as:

* OS and distro
* shell type
* installed tools
* package manager
* Git repo state
* Kubernetes context
* Docker/Podman/containerd availability

### 4. Knowledge and capability registry

Maps tool families to:

* supported commands
* validation sources
* troubleshooting templates
* risk classification rules

### 5. Local evidence collector

Queries system-native sources:

* `command -v`
* `type`
* `help`
* `<cmd> --help`
* `man`
* `whatis`
* `apropos`
* completion scripts
* package metadata
* `/usr/share/doc`

### 6. Suggestion engine

Builds one or more candidate commands using:

* deterministic mappings
* validated local flags
* environment-specific variants
* model assistance when needed

### 7. Troubleshooting engine

Given a symptom, produces:

* likely causes
* first safe checks
* decision tree
* next-step suggestions

### 8. Safety engine

Responsible for:

* risk labeling
* destructive command detection
* warning banners
* safe alternatives or dry-run preference

### 9. Logging and runbook engine

Stores:

* user query
* command suggestions
* chosen path
* notes
* optional final runbook summary

---

## 16. Suggested module layout

```text
cmd/
  cli-helper/
    main.go

internal/
  cli/
  parser/
  detect/
  context/
  evidence/
  registry/
  suggest/
  troubleshoot/
  safety/
  explain/
  audit/
  runbook/
  output/
  config/

pkg/
  models/
  utils/
```

### Example purpose of each module

* `parser/` → user intent parsing
* `detect/` → OS/tool discovery
* `context/` → Git/K8s/container runtime state collection
* `evidence/` → local documentation and flag extraction
* `registry/` → supported domains and commands
* `suggest/` → command generation
* `troubleshoot/` → guided diagnosis flows
* `safety/` → risk scoring and destructive detection
* `explain/` → flag and behavior explanations
* `audit/` → session logging
* `runbook/` → incident-style export
* `output/` → terminal rendering

---

## 17. Command verification strategy

## Modes

### 1. strict-local

Only recommend commands and flags verified from local system evidence.

### 2. local-preferred

Prefer local evidence, but allow model-assisted suggestions when evidence is incomplete.

### 3. explain-proof

Show command plus the exact source used for verification.

## Verification flow

1. Confirm command exists
2. Determine whether it is a binary, builtin, alias, or function
3. Parse local help/man/completion metadata
4. Extract valid flags or subcommands conservatively
5. Build candidate command
6. Assign confidence score
7. Render answer with explanation and risk

---

## 18. Safety model

## Risk classes

### Low risk

Read-only or informational commands
Examples:

* `git status`
* `ss -ltnp`
* `kubectl get pods`
* `df -h`

### Medium risk

Commands that change state but are usually recoverable
Examples:

* `systemctl restart nginx`
* `git reset HEAD~1`
* `docker restart container`

### High risk

Commands that can remove data, kill services, or alter production state significantly
Examples:

* `git reset --hard`
* `docker system prune -a`
* `kubectl delete`
* `rm -rf`

## Safety rules

* prefer read-only inspection first
* prefer dry-run variants where available
* never auto-run high-risk commands by default
* clearly explain potential impact
* require explicit confirmation before future execution features

---

## 19. Example wow features

### 1. Evidence-backed answers

Show verified source for commands and flags.

### 2. Explain like I am on call

Operationally focused explanations.

### 3. Troubleshooting trees

Decision-based guidance rather than one-off commands.

### 4. Read this output for me

Interpret pasted output and suggest next checks.

### 5. Runbook generation

Create documentation from command sessions.

### 6. Confidence scoring

Show confidence based on command presence, evidence availability, and context quality.

### 7. Environment-aware variants

Suggest distro- or tool-specific answers automatically.

---

## 20. Monetization options

## Option A: Free CLI + paid premium features

Free:

* core Linux and Git help
* basic explanation mode
* limited troubleshooting

Paid:

* Docker/containerd/Kubernetes/Terraform/Ansible packs
* advanced runbook export
* team audit features
* cloud integrations
* enterprise policy controls

## Option B: Open core

Core local CLI is open source.
Paid offerings include:

* enterprise support
* secure admin dashboard
* team knowledge packs
* controlled model backends
* compliance logging

## Option C: SaaS + local agent

Local CLI does discovery and safe guidance. Optional SaaS layer provides:

* sync of runbooks
* team knowledge sharing
* analytics on most common issues
* policy management

---

## 21. Ideal pricing concepts

### Individual plan

* affordable monthly or yearly plan
* advanced domains unlocked

### Team plan

* shared troubleshooting runbooks
* policy controls
* audit logs
* curated domain packs

### Enterprise plan

* local or self-hosted inference options
* compliance features
* private package and documentation ingestion
* support for secure environments

---

## 22. Competitive angle

The market already has command helpers and AI terminal assistants.

The opportunity is not to be just another command generator.

The opportunity is to be the best tool for users who care about:

* safety
* local verification
* explainability
* troubleshooting
* offline or privacy-friendly workflows
* Linux-first and operator-first design

Positioning:

**A safe, context-aware terminal copilot for operators and platform engineers.**

---

## 23. MVP delivery plan

## Stage 1: foundation

### Build

* CLI skeleton
* config system
* output renderer
* logging/audit base
* command existence detection
* local help/man parsing base

### Outcome

A working command-answer engine for a few Linux commands.

## Stage 2: Linux core support

### Build

* Linux core command registry
* explanation engine
* risk labeling
* expected output templates

### Outcome

Good support for files, processes, disk, services, networking basics.

## Stage 3: Git support

### Build

* Git intent pack
* Git repo state detection
* recovery and troubleshooting flows

### Outcome

Strong Git assistant for everyday workflows and panic recovery.

## Stage 4: troubleshooting engine

### Build

* service troubleshooting templates
* networking issue templates
* disk and process investigation flows
* output interpretation mode

### Outcome

The product starts to feel differentiated.

## Stage 5: packaging and early testers

### Build

* install scripts
* documentation
* examples
* feedback loop

### Outcome

Ready for alpha users.

---

## 24. Suggested roadmap timeline

## Part 1

* finalize product scope
* choose CLI UX
* build base architecture
* implement environment detection
* add Linux core answer mode

## Part 2

* add explanation mode
* add risk classification
* add systemd and networking packs
* add Git support

## Part 3

* add troubleshooting mode
* add output interpretation mode
* ship alpha release
* collect feedback

## Part 4

* refine based on real usage
* improve evidence parsing
* add archives/transfer/text-processing packs
* ship beta release

## Part 5+

* Docker
* containerd
* Kubernetes
* Terraform
* Ansible
* team and runbook features

---

## 25. Success metrics

### Product usage metrics

* number of queries per user per day
* percentage of users who return after first week
* most common domains used
* percentage of queries resolved without external searching

### Quality metrics

* command acceptance rate
* user-rated helpfulness
* local verification coverage rate
* hallucination/error rate
* false-risk classification rate

### Business metrics

* free-to-paid conversion
* team adoption
* active weekly users
* retention after 30 and 90 days

---

## 26. Risks and mitigation

### Risk 1: local parsing is messy

Some tools have inconsistent help output.

Mitigation:

* parse conservatively
* prefer fewer trustworthy suggestions over broad guessing
* use command-specific parsers for popular tools

### Risk 2: too much scope too early

Trying to support too many tools at once could slow delivery.

Mitigation:

* strict phased roadmap
* launch with Linux + Git first

### Risk 3: incorrect or risky suggestions

Bad suggestions could damage trust immediately.

Mitigation:

* strong safety engine
* read-only-first philosophy
* risk labels
* high-risk confirmation requirements

### Risk 4: product feels like a chatbot, not a serious tool

Mitigation:

* concise terminal-first UX
* evidence-backed output
* deterministic behavior where possible

---

## 27. Recommended tech stack

### Core language

* Go

### Why Go

* easy static binaries
* fast startup
* good CLI ecosystem
* easy cross-platform distribution
* strong support for integrations

### Possible libraries

* Cobra for CLI structure
* Viper for configuration
* Bubble Tea or simpler terminal rendering if richer UI is desired later
* structured logging library for audit and debugging

### Optional later integrations

* local model runners
* cloud LLM provider abstraction
* Kubernetes client-go
* Docker SDK

---

## 28. Naming directions

### Functional names

* CommandMate
* TermGuide
* OpsCLI
* ExplainCLI
* ShellGuide

### Operator-focused names

* RunbookCLI
* OpsPilot
* TerminalPilot
* GuardedCLI
* SafeShell

### Brandable names

* Nexishell
* ClarityOps
* Tersa
* VertaCLI
* Operon

---

## 29. Recommendation summary

The best version of this product is not a generic chatbot in the terminal.

It is a trusted operator assistant that:

* translates plain English into valid commands
* verifies what it can locally
* teaches the user what the command means
* helps troubleshoot step by step
* clearly separates safe inspection from risky action

## Recommended launch sequence

1. Linux core
2. systemd and networking
3. Git
4. troubleshooting engine
5. Docker, containerd, and Kubernetes
6. IaC and cloud tools

---

## 30. Immediate next steps

### Product decisions to lock now

* final product name
* MVP scope
* strict-local versus local-preferred default mode
* whether v1 includes troubleshoot mode or adds it in v1.5
* whether to open source core or keep fully commercial

### Build steps

1. scaffold Go CLI project
2. define command registry format
3. implement environment detection
4. implement local evidence collector
5. implement first 20 Linux intents
6. implement response renderer
7. implement risk model
8. implement Git pack
9. build alpha documentation and examples

### Local packaging foundations

Use the local scripts for reproducible workspace builds and installs:

* `scripts/build.sh`
* `scripts/test.sh`
* `scripts/install-local.sh`

Version output is available with:

* `cli-helper version`

---

## 31. Final statement

This product has real potential because it solves a daily pain point for technical users in a practical, explainable, and trustworthy way.

If executed well, it can become a terminal-native assistant that users rely on for speed, learning, and safer operations every day.
