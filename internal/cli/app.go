package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/noso-dev/noso/internal/audit"
	"github.com/noso-dev/noso/internal/config"
	"github.com/noso-dev/noso/internal/detect"
	"github.com/noso-dev/noso/internal/doctor"
	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/internal/history"
	"github.com/noso-dev/noso/internal/interpret"
	"github.com/noso-dev/noso/internal/output"
	"github.com/noso-dev/noso/internal/registry"
	"github.com/noso-dev/noso/internal/runbook"
	"github.com/noso-dev/noso/pkg/buildinfo"
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
	maxQueryBytes  = 64 * 1024        // 64 KB — plain-English questions are small
	maxInputBytes  = 512 * 1024       // 512 KB — for pasted command output in interpret mode
)

const usageText = `noso translates plain-English Linux and DevOps questions into safer command guidance.

Usage:
  cli-helper [flags] <plain-English question>
  cli-helper ask [flags] <plain-English question>
  cli-helper env
  cli-helper doctor
  cli-helper history [--limit N] [--match TEXT]
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
    local subcommands="ask env doctor history runbook version completion interpret"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${subcommands}" -- "${cur}") )
        return 0
    fi

    case "${prev}" in
        completion) COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") ) ;;
        --format)   COMPREPLY=( $(compgen -W "markdown json" -- "${cur}") ) ;;
        --output)   COMPREPLY=( $(compgen -f -- "${cur}") ) ;;
    esac
}
complete -F _cli_helper cli-helper
`

const zshCompletion = `#compdef cli-helper
_cli_helper() {
    local -a subcommands
    subcommands=(ask env doctor history runbook version completion interpret)
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
set -l subcommands ask env doctor history runbook version completion interpret
complete -c cli-helper -f -n "not __fish_seen_subcommand_from $subcommands" -a "$subcommands"
complete -c cli-helper -f -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"
`
