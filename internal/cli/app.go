package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/NdumLab/noso/internal/audit"
	"github.com/NdumLab/noso/internal/config"
	"github.com/NdumLab/noso/internal/detect"
	"github.com/NdumLab/noso/internal/doctor"
	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/history"
	"github.com/NdumLab/noso/internal/incident"
	"github.com/NdumLab/noso/internal/interpret"
	"github.com/NdumLab/noso/internal/llm"
	"github.com/NdumLab/noso/internal/output"
	"github.com/NdumLab/noso/internal/registry"
	"github.com/NdumLab/noso/internal/runbook"
	"github.com/NdumLab/noso/internal/troubleshoot"
	"github.com/NdumLab/noso/pkg/buildinfo"
	"github.com/NdumLab/noso/pkg/models"
)

// Exit codes used by the CLI.  Scripts can branch on these.
const (
	ExitOK          = 0 // success
	ExitErr         = 1 // internal / config error
	ExitUsage       = 2 // bad arguments or missing required flags
	ExitNoIntent    = 3 // query received but no intent matched
	ExitToolMissing = 4 // required tool not found on this host (reserved)
)

// Input size guards prevent the CLI from consuming unbounded memory on
// accidental pipe-ins or malformed invocations.
const (
	maxQueryBytes = 64 * 1024  // 64 KB — plain-English questions are small
	maxInputBytes = 512 * 1024 // 512 KB — for pasted command output in interpret mode
)

const usageText = `noso translates plain-English Linux and DevOps questions into safer command guidance.

Usage:
  cli-helper [flags] <plain-English question>
  cli-helper ask [flags] <plain-English question>
  cli-helper env
  cli-helper doctor
  cli-helper history [--limit N] [--match TEXT]
  cli-helper incident-status [--query TEXT]
  cli-helper incident-history [--limit N] [--query TEXT] [--match TEXT] [--status open|resolved]
  cli-helper incident-ingest --query TEXT [--source TEXT] [--severity TEXT] [--summary TEXT] [--fingerprint TEXT] [--label key=value]
  cli-helper incident-observe --query TEXT [--max-steps N]
  cli-helper incident-resolve --query TEXT [--summary TEXT]
  cli-helper llm-log [--limit N] [--match TEXT] [--since RFC3339|DURATION] [--provider NAME] [--error-only] [--clarification-only] [--stats] [--format text|json|markdown|csv] [--output PATH]
  cli-helper troubleshoot [flags] <plain-English question>
  cli-helper troubleshoot-history [--limit N] [--query TEXT] [--match TEXT]
  cli-helper troubleshoot-reset [--query TEXT]
  cli-helper runbook [--limit N] [--match TEXT] [--format markdown|json] [--output PATH]
  cli-helper version
  cli-helper completion <bash|zsh|fish>
  cli-helper interpret --command "<command>" [--input "<captured output>"]

Examples:
  cli-helper "what process is using port 8080"
  cli-helper "explain git reset --hard HEAD~1"
  cli-helper ask "show disk usage of /var"
  cli-helper env
  cli-helper doctor
  cli-helper history --limit 5 --match git
  cli-helper incident-status --query "why is worker 2 not up?"
  cli-helper incident-history --status open
  cli-helper incident-ingest --query "api availability alert" --source alertmanager --severity critical --summary "API error rate above threshold" --label service=api --label namespace=prod
  cli-helper incident-observe --query "why is worker 2 not up?"
  cli-helper incident-observe --query "why is worker 2 not up?" --max-steps 3
  cli-helper incident-resolve --query "why is worker 2 not up?" --summary "Pod image pull secret fixed"
  cli-helper llm-log --limit 10 --match timeout
  cli-helper llm-log --since 2h --match transient
  cli-helper llm-log --provider ollama --error-only
  cli-helper llm-log --clarification-only
  cli-helper llm-log --since 24h --stats
  cli-helper troubleshoot "why is worker 2 not up?"
  cli-helper troubleshoot-history --query "why is worker 2 not up?"
  cli-helper troubleshoot-reset --query "why is worker 2 not up?"
  cli-helper llm-log --since 24h --stats --format markdown --output llm-summary.md
  cli-helper llm-log --provider ollama --error-only --format csv --output llm-errors.csv
  cli-helper runbook --limit 10 --format markdown --output incident.md
  cli-helper version
  cli-helper completion bash > /etc/bash_completion.d/cli-helper
  cli-helper interpret --command "df -h" --input "Filesystem Size Used Avail Use% Mounted on"

Flags:
`

func Run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (int, error) {
	fs := flag.NewFlagSet("cli-helper", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var jsonOut bool
	var quiet bool
	var debug bool
	fs.BoolVar(&jsonOut, "json", false, "render the answer as JSON")
	fs.BoolVar(&quiet, "quiet", false, "suppress warnings and informational output")
	fs.BoolVar(&debug, "debug", false, "write debug information to stderr")
	fs.Usage = func() {
		fmt.Fprint(stderr, usageText)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK, nil
		}
		return ExitUsage, err
	}

	cfg, err := config.Load()
	if err != nil {
		return ExitErr, fmt.Errorf("config load failed: %w", err)
	}

	env, err := detect.Local()
	if err != nil {
		return ExitErr, fmt.Errorf("environment detection failed: %w", err)
	}

	if debug {
		fmt.Fprintf(stderr, "[debug] distro=%s pkg=%s rhel9=%v\n",
			env.Distro, env.PackageManager, env.IsRHEL9)
	}

	collector := evidence.NewCollector()
	logger := audit.NewLogger(cfg.AuditLogPath)

	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return ExitUsage, nil
	}

	switch rest[0] {
	case "ask":
		rest = rest[1:]
	case "env":
		rendered, renderErr := output.RenderEnvironment(env, jsonOut)
		if renderErr != nil {
			return ExitErr, renderErr
		}
		fmt.Fprint(stdout, rendered)
		return ExitOK, nil
	case "doctor":
		response := doctor.Check(cfg, env)
		rendered, renderErr := output.RenderResponse(response, jsonOut, quiet)
		if renderErr != nil {
			return ExitErr, renderErr
		}
		fmt.Fprint(stdout, rendered)
		return ExitOK, nil
	case "history":
		return runHistory(rest[1:], stdout, stderr, jsonOut, quiet, cfg)
	case "incident-status":
		return runIncidentStatus(rest[1:], stdout, stderr, jsonOut, cfg)
	case "incident-history":
		return runIncidentHistory(rest[1:], stdout, stderr, jsonOut, cfg)
	case "incident-ingest":
		return runIncidentIngest(rest[1:], stdout, stderr, jsonOut, cfg)
	case "incident-observe":
		return runIncidentObserve(rest[1:], stdout, stderr, jsonOut, quiet, cfg)
	case "incident-resolve":
		return runIncidentResolve(rest[1:], stdout, stderr, cfg)
	case "llm-log":
		return runLLMLog(rest[1:], stdout, stderr, jsonOut, cfg)
	case "troubleshoot":
		return runTroubleshoot(rest[1:], stdout, stderr, jsonOut, quiet, cfg, env, collector, logger)
	case "troubleshoot-history":
		return runTroubleshootHistory(rest[1:], stdout, stderr, jsonOut, cfg)
	case "troubleshoot-reset":
		return runTroubleshootReset(rest[1:], stdout, stderr, cfg)
	case "runbook":
		return runRunbook(rest[1:], stdout, stderr, cfg)
	case "version":
		fmt.Fprintln(stdout, buildinfo.String())
		return ExitOK, nil
	case "completion":
		return runCompletion(rest[1:], stdout, stderr)
	case "interpret":
		return runInterpret(rest[1:], stdin, stdout, stderr, jsonOut, quiet)
	}

	if len(rest) == 0 {
		return ExitUsage, fmt.Errorf("missing plain-English question")
	}

	query := strings.Join(rest, " ")
	if len(query) > maxQueryBytes {
		return ExitUsage, fmt.Errorf("query too long: %d bytes (max %d)", len(query), maxQueryBytes)
	}

	response, err := registry.Resolve(query, env, collector)
	if err != nil {
		return ExitErr, err
	}

	if response.IntentID == "unsupported_query" {
		if plan, ok, planErr := registry.TroubleshootPlan(query, env, collector); planErr == nil && ok {
			response = plan
		} else if planErr != nil {
			return ExitErr, planErr
		}
	}

	if response.IntentID == "unsupported_query" && cfg.LLMEnabled {
		if fallback, ok, fallbackErr := resolveWithLLM(cfg, query, env, collector); fallbackErr == nil && ok {
			response = fallback
		} else if fallbackErr != nil && !quiet {
			response.Warnings = append(response.Warnings, llm.DescribeError(fallbackErr)+": "+fallbackErr.Error())
		}
	}

	if err := logger.Append(query, response); err != nil && !quiet {
		response.Warnings = append(response.Warnings, "audit log unavailable: "+err.Error())
	}

	exitCode := ExitOK
	if response.IntentID == "unsupported_query" {
		exitCode = ExitNoIntent
	}

	rendered, err := output.RenderResponse(response, jsonOut, quiet)
	if err != nil {
		return ExitErr, err
	}

	fmt.Fprint(stdout, rendered)
	return exitCode, nil
}

func resolveWithLLM(cfg config.Config, query string, env models.Environment, collector evidence.Collector) (models.Response, bool, error) {
	client := llm.NewClient(cfg)
	req := llm.BuildRequest(query, env)
	resp, err := client.Interpret(context.Background(), req)
	if err != nil {
		return models.Response{}, false, err
	}

	if resp.NeedsClarification {
		if plan, ok, err := registry.TroubleshootPlanFromCandidates(query, env, collector, resp.Candidates); err == nil && ok {
			plan.Warnings = append(plan.Warnings, "clarification prompt from local llm: "+resp.ClarificationQuestion)
			return plan, true, nil
		} else if err != nil {
			return models.Response{}, false, err
		}
		return registry.ClarificationResponse(resp.ClarificationQuestion, resp.Candidates), true, nil
	}

	for _, candidate := range llm.RankedCandidates(resp, 0.5) {
		resolved, ok, resolveErr := registry.ResolveLLMCandidate(candidate, env, collector)
		if resolveErr != nil {
			return models.Response{}, false, resolveErr
		}
		if ok {
			return resolved, true, nil
		}
	}

	return models.Response{}, false, nil
}

func runTroubleshoot(args []string, stdout io.Writer, stderr io.Writer, jsonOut, quiet bool, cfg config.Config, env models.Environment, collector evidence.Collector, logger audit.Logger) (int, error) {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return ExitUsage, fmt.Errorf("troubleshoot requires a plain-English question")
	}
	if len(query) > maxQueryBytes {
		return ExitUsage, fmt.Errorf("query too long: %d bytes (max %d)", len(query), maxQueryBytes)
	}

	state, stateErr := troubleshoot.LoadState(cfg.TroubleshootStatePath)
	if stateErr != nil && !quiet {
		stateErr = fmt.Errorf("troubleshoot state unavailable: %w", stateErr)
	}
	planningQuery := query
	threadKey := query
	var threadContext troubleshoot.StateThread
	var clarificationHint troubleshoot.ClarificationHint
	var clarified bool
	var adoptedTarget string
	var directResponse models.Response
	var directResponseOK bool
	if stateErr == nil {
		if latestThread, suggestion, ok := troubleshoot.ResolveSuggestedTarget(state, query); ok {
			planningQuery = troubleshoot.ApplySuggestedTargetQuery(latestThread.Query, suggestion)
			threadKey = latestThread.Query
			threadContext = troubleshoot.ApplySuggestedTarget(latestThread, suggestion)
			adoptedTarget = formatSuggestedTarget(suggestion)
			if response, ok := troubleshoot.SuggestedTargetResponse(suggestion); ok {
				directResponse = response
				directResponseOK = true
			}
			state = troubleshoot.UpdateThread(state, threadKey, threadContext)
		} else if latestThread, hint, ok := troubleshoot.ResolveClarification(state, query); ok {
			clarified = true
			clarificationHint = hint
			planningQuery = troubleshoot.ApplyClarificationQuery(latestThread.Query, hint)
			threadKey = latestThread.Query
			threadContext = troubleshoot.ApplyClarificationHint(latestThread, hint)
			state = troubleshoot.UpdateThread(state, threadKey, threadContext)
		}
		if latestThread, refinement, ok := troubleshoot.ResolveThreadRefinement(state, query); ok {
			threadKey = latestThread.Query
			threadContext = troubleshoot.ApplyThreadRefinement(latestThread, refinement)
			planningQuery = troubleshoot.ApplyThreadRefinementQuery(planningQuery, threadContext, refinement)
			if threadContext.ActiveTarget != "" {
				adoptedTarget = formatAdoptedTarget(threadContext)
			}
			if response, ok := directResponseForThread(threadContext); ok {
				directResponse = response
				directResponseOK = true
			}
			state = troubleshoot.UpdateThread(state, threadKey, threadContext)
		}
	}

	var response models.Response
	var ok bool
	var err error
	if directResponseOK {
		response = directResponse
		ok = true
	} else {
		response, ok, err = registry.TroubleshootPlan(planningQuery, env, collector)
		if err != nil {
			return ExitErr, err
		}
		if !ok && cfg.LLMEnabled {
			response, ok, err = resolveWithLLM(cfg, planningQuery, env, collector)
			if err != nil {
				return ExitErr, err
			}
		}
	}
	if !ok {
		response = registry.ClarificationResponse(
			"noso could not confidently classify this outage yet. Name the object type if you can, for example: service, container, pod, host, or port.",
			nil,
		)
		response.IntentID = "troubleshoot_unclassified"
		response.ExpectedOutput = "A narrower troubleshoot question that names the failing service, container, pod, or host."
	} else {
		var existing troubleshoot.StateThread
		if stateErr == nil {
			if thread, found := troubleshoot.FindThread(state, threadKey); found {
				existing = thread
				response = troubleshoot.ApplyThreadContext(response, thread)
			}
		}
		response = troubleshoot.EnrichWithLiveEvidence(response)
		specializeThread := troubleshoot.PreviewThread(existing, threadKey, response)
		if stateErr == nil {
			response = troubleshoot.SpecializeInfrastructureProbes(response, env, specializeThread)
		} else {
			response = troubleshoot.SpecializeInfrastructureProbes(response, env, troubleshoot.StateThread{})
		}
		if adoptedTarget != "" {
			response.AdoptedTarget = adoptedTarget
		}
	}
	if stateErr == nil {
		var existing troubleshoot.StateThread
		if thread, found := troubleshoot.FindThread(state, threadKey); found {
			existing = thread
		}
		finalThread := troubleshoot.PreviewThread(existing, threadKey, response)
		response = troubleshoot.AttachLikelyCauses(response, finalThread)
		if clarified {
			response.Warnings = append(response.Warnings, "applied operator clarification: treating the current thread as a "+clarificationHint.Label+" problem")
		}
		if saveErr := troubleshoot.SaveState(cfg.TroubleshootStatePath, troubleshoot.UpdateState(state, threadKey, response)); saveErr != nil && !quiet {
			response.Warnings = append(response.Warnings, "troubleshoot state save failed: "+saveErr.Error())
		}
		if incidentState, err := incident.LoadState(cfg.IncidentStatePath); err == nil {
			incidentState = incident.UpdateFromTroubleshoot(incidentState, finalThread, response)
			if saveErr := incident.SaveState(cfg.IncidentStatePath, incidentState); saveErr != nil && !quiet {
				response.Warnings = append(response.Warnings, "incident state save failed: "+saveErr.Error())
			}
		} else if !quiet {
			response.Warnings = append(response.Warnings, "incident state unavailable: "+err.Error())
		}
	} else if !quiet {
		response.Warnings = append(response.Warnings, stateErr.Error())
	}

	if err := logger.Append(query, response); err != nil && !quiet {
		response.Warnings = append(response.Warnings, "audit log unavailable: "+err.Error())
	}
	rendered, err := output.RenderResponse(response, jsonOut, quiet)
	if err != nil {
		return ExitErr, err
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func runHistory(args []string, stdout io.Writer, stderr io.Writer, jsonOut, quiet bool, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var limit int
	var match string
	fs.IntVar(&limit, "limit", 10, "maximum number of history entries to show")
	fs.StringVar(&match, "match", "", "filter history by query, intent, or command text")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK, nil
		}
		return ExitUsage, err
	}

	records, err := audit.ReadAll(cfg.AuditLogPath)
	if err != nil {
		return ExitErr, fmt.Errorf("history read failed: %w", err)
	}
	records = audit.Filter(records, match, limit)

	rendered, err := history.Render(records, jsonOut)
	if err != nil {
		return ExitErr, err
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func runIncidentStatus(args []string, stdout io.Writer, stderr io.Writer, jsonOut bool, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("incident-status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	query := fs.String("query", "", "incident query to inspect")
	if err := fs.Parse(args); err != nil {
		return ExitUsage, err
	}

	state, err := incident.LoadState(cfg.IncidentStatePath)
	if err != nil {
		return ExitErr, err
	}
	if strings.TrimSpace(*query) == "" && len(state.Incidents) > 0 {
		rendered, renderErr := incident.RenderStatus(state.Incidents[0], jsonOut)
		if renderErr != nil {
			return ExitErr, renderErr
		}
		fmt.Fprint(stdout, rendered)
		return ExitOK, nil
	}
	record, _ := incident.Find(state, *query)
	rendered, renderErr := incident.RenderStatus(record, jsonOut)
	if renderErr != nil {
		return ExitErr, renderErr
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func runIncidentHistory(args []string, stdout io.Writer, stderr io.Writer, jsonOut bool, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("incident-history", flag.ContinueOnError)
	fs.SetOutput(stderr)
	limit := fs.Int("limit", 20, "maximum incidents to show")
	query := fs.String("query", "", "exact incident query to match")
	match := fs.String("match", "", "substring to match across incident content")
	status := fs.String("status", "", "incident status filter: open or resolved")
	if err := fs.Parse(args); err != nil {
		return ExitUsage, err
	}
	state, err := incident.LoadState(cfg.IncidentStatePath)
	if err != nil {
		return ExitErr, err
	}
	records := incident.Filter(state.Incidents, *query, *match, *status, *limit)
	rendered, renderErr := incident.RenderHistory(records, jsonOut)
	if renderErr != nil {
		return ExitErr, renderErr
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

type labelsFlag map[string]string

func (f *labelsFlag) String() string {
	if f == nil || len(*f) == 0 {
		return ""
	}
	var parts []string
	for key, value := range *f {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
}

func (f *labelsFlag) Set(value string) error {
	parts := strings.SplitN(strings.TrimSpace(value), "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return fmt.Errorf("label must be key=value")
	}
	if *f == nil {
		*f = map[string]string{}
	}
	(*f)[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	return nil
}

func runIncidentIngest(args []string, stdout io.Writer, stderr io.Writer, jsonOut bool, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("incident-ingest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	query := fs.String("query", "", "incident query or title to open/update")
	source := fs.String("source", "", "alert source, such as alertmanager")
	severity := fs.String("severity", "", "alert severity, such as critical or warning")
	summary := fs.String("summary", "", "alert summary text")
	fingerprint := fs.String("fingerprint", "", "stable alert fingerprint for deduplication")
	var labels labelsFlag
	fs.Var(&labels, "label", "alert label in key=value form; repeat for multiple labels")
	if err := fs.Parse(args); err != nil {
		return ExitUsage, err
	}
	if strings.TrimSpace(*query) == "" && strings.TrimSpace(*summary) == "" {
		return ExitUsage, fmt.Errorf("incident-ingest requires --query or --summary")
	}
	state, err := incident.LoadState(cfg.IncidentStatePath)
	if err != nil {
		return ExitErr, err
	}
	alert := incident.Alert{
		Query:       strings.TrimSpace(*query),
		Source:      strings.TrimSpace(*source),
		Severity:    strings.TrimSpace(*severity),
		Summary:     strings.TrimSpace(*summary),
		Fingerprint: strings.TrimSpace(*fingerprint),
		Labels:      map[string]string(labels),
	}
	state = incident.UpsertAlert(state, alert)
	if err := incident.SaveState(cfg.IncidentStatePath, state); err != nil {
		return ExitErr, err
	}
	record, _ := incident.Find(state, firstCLIValue(alert.Query, alert.Summary))
	rendered, renderErr := incident.RenderStatus(record, jsonOut)
	if renderErr != nil {
		return ExitErr, renderErr
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func runIncidentResolve(args []string, stdout io.Writer, stderr io.Writer, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("incident-resolve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	query := fs.String("query", "", "incident query to resolve")
	summary := fs.String("summary", "", "resolution summary")
	if err := fs.Parse(args); err != nil {
		return ExitUsage, err
	}
	if strings.TrimSpace(*query) == "" {
		return ExitUsage, fmt.Errorf("incident-resolve requires --query")
	}
	state, err := incident.LoadState(cfg.IncidentStatePath)
	if err != nil {
		return ExitErr, err
	}
	state = incident.Resolve(state, *query, *summary)
	if err := incident.SaveState(cfg.IncidentStatePath, state); err != nil {
		return ExitErr, err
	}
	fmt.Fprintf(stdout, "Resolved incident: %s\n", strings.TrimSpace(*query))
	return ExitOK, nil
}

func runIncidentObserve(args []string, stdout io.Writer, stderr io.Writer, jsonOut, quiet bool, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("incident-observe", flag.ContinueOnError)
	fs.SetOutput(stderr)
	query := fs.String("query", "", "incident query to observe")
	maxSteps := fs.Int("max-steps", 1, "maximum approved read-only probes to run in sequence")
	if err := fs.Parse(args); err != nil {
		return ExitUsage, err
	}
	if strings.TrimSpace(*query) == "" {
		return ExitUsage, fmt.Errorf("incident-observe requires --query")
	}
	if *maxSteps <= 0 {
		return ExitUsage, fmt.Errorf("incident-observe requires --max-steps >= 1")
	}

	incidentState, err := incident.LoadState(cfg.IncidentStatePath)
	if err != nil {
		return ExitErr, err
	}
	record, ok := incident.Find(incidentState, *query)
	if !ok {
		return ExitUsage, fmt.Errorf("incident not found for query %q", strings.TrimSpace(*query))
	}

	responses, _, err := incident.ObserveMany(record, *maxSteps)
	if err != nil {
		return ExitErr, err
	}
	if len(responses) == 0 {
		return ExitErr, fmt.Errorf("incident observe produced no responses")
	}
	troubleshootState, stateErr := troubleshoot.LoadState(cfg.TroubleshootStatePath)
	var finalResponse models.Response
	var thread troubleshoot.StateThread
	for idx, response := range responses {
		if stateErr == nil {
			if existing, found := troubleshoot.FindThread(troubleshootState, record.Query); found {
				thread = troubleshoot.PreviewThread(existing, record.Query, response)
			}
		}
		if thread.Query == "" {
			thread = troubleshoot.StateThread{
				Query:           record.Query,
				ActiveFamily:    record.ActiveFamily,
				ActiveTarget:    record.ActiveTarget,
				ActiveNamespace: record.Namespace,
			}
			thread = troubleshoot.PreviewThread(thread, record.Query, response)
		}
		response = troubleshoot.AttachLikelyCauses(response, thread)
		if idx > 0 {
			response.Warnings = append(response.Warnings, fmt.Sprintf("incident observe advanced through %d approved probes in this run", idx+1))
		}
		if stateErr == nil {
			troubleshootState = troubleshoot.UpdateState(troubleshootState, record.Query, response)
		}
		incidentState = incident.UpdateFromTroubleshoot(incidentState, thread, response)
		finalResponse = response
	}
	if stateErr == nil {
		if saveErr := troubleshoot.SaveState(cfg.TroubleshootStatePath, troubleshootState); saveErr != nil && !quiet {
			finalResponse.Warnings = append(finalResponse.Warnings, "troubleshoot state save failed: "+saveErr.Error())
		}
	} else if !quiet {
		finalResponse.Warnings = append(finalResponse.Warnings, "troubleshoot state unavailable: "+stateErr.Error())
	}
	if saveErr := incident.SaveState(cfg.IncidentStatePath, incidentState); saveErr != nil && !quiet {
		finalResponse.Warnings = append(finalResponse.Warnings, "incident state save failed: "+saveErr.Error())
	}

	rendered, renderErr := output.RenderResponse(finalResponse, jsonOut, quiet)
	if renderErr != nil {
		return ExitErr, renderErr
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func formatAdoptedTarget(thread troubleshoot.StateThread) string {
	if thread.ActiveTarget == "" || thread.ActiveFamily == "" {
		return ""
	}
	parts := []string{thread.ActiveFamily}
	if thread.ActiveNamespace != "" {
		parts = append(parts, "namespace "+thread.ActiveNamespace)
	}
	if thread.RuntimeHint != "" {
		parts = append(parts, thread.RuntimeHint)
	}
	return thread.ActiveTarget + " (" + strings.Join(parts, ", ") + ")"
}

func formatSuggestedTarget(suggestion troubleshoot.SuggestedTarget) string {
	if suggestion.Name == "" || suggestion.Family == "" {
		return ""
	}
	parts := []string{suggestion.Family}
	if suggestion.Namespace != "" {
		parts = append(parts, "namespace "+suggestion.Namespace)
	}
	return suggestion.Name + " (" + strings.Join(parts, ", ") + ")"
}

func directResponseForThread(thread troubleshoot.StateThread) (models.Response, bool) {
	if thread.ActiveTarget == "" || thread.ActiveFamily == "" {
		return models.Response{}, false
	}
	var command string
	switch thread.ActiveFamily {
	case "kubernetes-pvc":
		command = "kubectl describe pvc " + thread.ActiveTarget
	case "kubernetes-secret":
		command = "kubectl describe secret " + thread.ActiveTarget
	case "kubernetes-configmap":
		command = "kubectl describe configmap " + thread.ActiveTarget
	case "kubernetes-deployment":
		command = "kubectl describe deployment " + thread.ActiveTarget
	case "kubernetes-service":
		command = "kubectl describe service " + thread.ActiveTarget
	case "kubernetes-node":
		command = "kubectl describe node " + thread.ActiveTarget
	default:
		return models.Response{}, false
	}
	if thread.ActiveNamespace != "" && thread.ActiveFamily != "kubernetes-node" {
		fields := strings.Fields(command)
		if len(fields) >= 4 {
			command = strings.Join([]string{fields[0], fields[1], fields[2], "-n", thread.ActiveNamespace, fields[3]}, " ")
		}
	}
	return troubleshoot.SuggestedTargetResponse(troubleshoot.SuggestedTarget{
		Family:    thread.ActiveFamily,
		Name:      thread.ActiveTarget,
		Namespace: thread.ActiveNamespace,
		Command:   command,
	})
}

func firstCLIValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func runLLMLog(args []string, stdout io.Writer, stderr io.Writer, jsonOut bool, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("llm-log", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var limit int
	var match string
	var sinceArg string
	var provider string
	var errorOnly bool
	var clarificationOnly bool
	var stats bool
	var format string
	var outputPath string
	fs.IntVar(&limit, "limit", 10, "maximum number of LLM log entries to show")
	fs.StringVar(&match, "match", "", "filter LLM log entries by query, provider, model, intent, or error text")
	fs.StringVar(&sinceArg, "since", "", "show only LLM log entries since an RFC3339 timestamp or duration like 15m or 2h")
	fs.StringVar(&provider, "provider", "", "show only LLM log entries for a specific provider such as heuristic or ollama")
	fs.BoolVar(&errorOnly, "error-only", false, "show only LLM log entries that recorded an error")
	fs.BoolVar(&clarificationOnly, "clarification-only", false, "show only LLM log entries that required clarification")
	fs.BoolVar(&stats, "stats", false, "render an aggregate summary instead of raw LLM log entries")
	fs.StringVar(&format, "format", "", "render format: text, json, markdown, or csv (defaults to text or json when --json is set)")
	fs.StringVar(&outputPath, "output", "", "optional output path for rendered LLM log entries or summaries")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK, nil
		}
		return ExitUsage, err
	}

	if cfg.LLMLogPath == "" {
		return ExitErr, fmt.Errorf("llm log path is not configured; set NOSO_LLM_LOG_PATH or llm_log_path")
	}

	records, err := llm.ReadLog(cfg.LLMLogPath)
	if err != nil {
		return ExitErr, fmt.Errorf("llm log read failed: %w", err)
	}
	since, err := llm.ParseSince(sinceArg, time.Now().UTC())
	if err != nil {
		return ExitUsage, err
	}
	records = llm.FilterLogsWith(records, llm.LogFilter{
		Match:             match,
		Limit:             limit,
		Since:             since,
		Provider:          provider,
		ErrorOnly:         errorOnly,
		ClarificationOnly: clarificationOnly,
	})

	if strings.TrimSpace(format) == "" {
		if jsonOut {
			format = "json"
		} else {
			format = "text"
		}
	}

	var rendered string
	if stats {
		rendered, err = llm.RenderLogSummaryFormat(llm.SummarizeLogs(records), format)
	} else {
		rendered, err = llm.RenderLogsFormat(records, format)
	}
	if err != nil {
		return ExitErr, err
	}
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(rendered), 0o600); err != nil {
			return ExitErr, fmt.Errorf("llm log write failed: %w", err)
		}
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func runTroubleshootHistory(args []string, stdout io.Writer, stderr io.Writer, jsonOut bool, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("troubleshoot-history", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var limit int
	var query string
	var match string
	fs.IntVar(&limit, "limit", 10, "maximum number of troubleshoot probe entries to show")
	fs.StringVar(&query, "query", "", "show only troubleshoot history for the matching plain-English question")
	fs.StringVar(&match, "match", "", "filter troubleshoot history by query, command, summary, findings, or warnings")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK, nil
		}
		return ExitUsage, err
	}

	state, err := troubleshoot.LoadState(cfg.TroubleshootStatePath)
	if err != nil {
		return ExitErr, fmt.Errorf("troubleshoot state read failed: %w", err)
	}
	records := troubleshoot.FilterHistory(troubleshoot.FlattenHistory(state), query, match, limit)
	rendered, err := troubleshoot.RenderHistory(records, jsonOut)
	if err != nil {
		return ExitErr, err
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func runTroubleshootReset(args []string, stdout io.Writer, stderr io.Writer, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("troubleshoot-reset", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var query string
	fs.StringVar(&query, "query", "", "reset troubleshoot state only for the matching plain-English question; omit to clear all troubleshoot state")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK, nil
		}
		return ExitUsage, err
	}

	state, err := troubleshoot.LoadState(cfg.TroubleshootStatePath)
	if err != nil {
		return ExitErr, fmt.Errorf("troubleshoot state read failed: %w", err)
	}
	state = troubleshoot.ResetState(state, query)
	if err := troubleshoot.SaveState(cfg.TroubleshootStatePath, state); err != nil {
		return ExitErr, fmt.Errorf("troubleshoot state reset failed: %w", err)
	}
	if strings.TrimSpace(query) == "" {
		fmt.Fprintln(stdout, "Cleared all troubleshoot state.")
	} else {
		fmt.Fprintf(stdout, "Cleared troubleshoot state for query: %s\n", query)
	}
	return ExitOK, nil
}

func runRunbook(args []string, stdout io.Writer, stderr io.Writer, cfg config.Config) (int, error) {
	fs := flag.NewFlagSet("runbook", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var limit int
	var match string
	var format string
	var outputPath string
	fs.IntVar(&limit, "limit", 10, "maximum number of audit entries to include")
	fs.StringVar(&match, "match", "", "filter runbook records by query, intent, or command text")
	fs.StringVar(&format, "format", "markdown", "render format: markdown or json")
	fs.StringVar(&outputPath, "output", "", "optional output path for the rendered runbook")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK, nil
		}
		return ExitUsage, err
	}

	records, err := audit.ReadAll(cfg.AuditLogPath)
	if err != nil {
		return ExitErr, fmt.Errorf("runbook read failed: %w", err)
	}
	records = audit.Filter(records, match, limit)

	report := runbook.Build(records)
	rendered, err := runbook.Render(report, format)
	if err != nil {
		return ExitErr, err
	}
	if outputPath != "" {
		// 0o600: runbooks may contain sensitive command history.
		if err := os.WriteFile(outputPath, []byte(rendered), 0o600); err != nil {
			return ExitErr, fmt.Errorf("runbook write failed: %w", err)
		}
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func runInterpret(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, jsonOut, quiet bool) (int, error) {
	fs := flag.NewFlagSet("interpret", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var command string
	var input string
	fs.StringVar(&command, "command", "", "command that produced the captured output")
	fs.StringVar(&input, "input", "", "captured command output to interpret")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK, nil
		}
		return ExitUsage, err
	}
	if strings.TrimSpace(command) == "" {
		return ExitUsage, fmt.Errorf("interpret requires --command")
	}

	if strings.TrimSpace(input) == "" {
		data, err := io.ReadAll(io.LimitReader(stdin, maxInputBytes+1))
		if err != nil {
			return ExitErr, fmt.Errorf("interpret input read failed: %w", err)
		}
		if len(data) > maxInputBytes {
			return ExitUsage, fmt.Errorf("interpret input too large: max %d bytes", maxInputBytes)
		}
		input = string(data)
	} else if len(input) > maxInputBytes {
		return ExitUsage, fmt.Errorf("interpret input too large: max %d bytes", maxInputBytes)
	}

	response, err := interpret.Output(command, input)
	if err != nil {
		return ExitErr, err
	}

	rendered, err := output.RenderResponse(response, jsonOut, quiet)
	if err != nil {
		return ExitErr, err
	}
	fmt.Fprint(stdout, rendered)
	return ExitOK, nil
}

func runCompletion(args []string, stdout io.Writer, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: cli-helper completion <bash|zsh|fish>")
		return ExitUsage, nil
	}
	switch args[0] {
	case "bash":
		fmt.Fprint(stdout, bashCompletion)
	case "zsh":
		fmt.Fprint(stdout, zshCompletion)
	case "fish":
		fmt.Fprint(stdout, fishCompletion)
	default:
		fmt.Fprintf(stderr, "unknown shell %q: supported shells are bash, zsh, fish\n", args[0])
		return ExitUsage, nil
	}
	return ExitOK, nil
}

const bashCompletion = `# bash completion for cli-helper
_cli_helper() {
    local cur prev words
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    local subcommands="ask env doctor history incident-status incident-history incident-ingest incident-observe incident-resolve llm-log troubleshoot troubleshoot-history troubleshoot-reset runbook version completion interpret"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${subcommands}" -- "${cur}") )
        return 0
    fi

    case "${prev}" in
        completion) COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") ) ;;
        --format)   COMPREPLY=( $(compgen -W "text markdown json csv" -- "${cur}") ) ;;
        --output)   COMPREPLY=( $(compgen -f -- "${cur}") ) ;;
    esac
}
complete -F _cli_helper cli-helper
`

const zshCompletion = `#compdef cli-helper
_cli_helper() {
    local -a subcommands
    subcommands=(ask env doctor history incident-status incident-history incident-ingest incident-observe incident-resolve llm-log troubleshoot troubleshoot-history troubleshoot-reset runbook version completion interpret)
    _arguments \
        '1: :->subcmd' \
        '*: :->args'
    case $state in
        subcmd) _describe 'subcommand' subcommands ;;
        args)
            case $words[2] in
                completion) _values 'shell' bash zsh fish ;;
            esac
            ;;
    esac
}
_cli_helper
`

const fishCompletion = `# fish completion for cli-helper
set -l subcommands ask env doctor history incident-status incident-history incident-ingest incident-observe incident-resolve llm-log troubleshoot troubleshoot-history troubleshoot-reset runbook version completion interpret
complete -c cli-helper -f -n "not __fish_seen_subcommand_from $subcommands" -a "$subcommands"
complete -c cli-helper -f -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"
`
