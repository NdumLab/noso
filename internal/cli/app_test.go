package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/llm"
	"github.com/NdumLab/noso/internal/troubleshoot"
	"github.com/NdumLab/noso/pkg/models"
)

func TestRunInterpretMode(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	input := strings.NewReader("Filesystem Size Used Avail Use% Mounted on\n/dev/root 50G 47G 3G 94% /\n")

	code, err := Run([]string{"interpret", "--command", "df -h"}, input, stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "interpret_df") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunDoctorMode(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code, err := Run([]string{"doctor"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Intent: doctor") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunVersionMode(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code, err := Run([]string{"version"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "version=") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunHistoryMode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NOSO_AUDIT_LOG_PATH", dir+"/audit.log")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code, err := Run([]string{"show", "disk", "free", "space"}, strings.NewReader(""), stdout, stderr)
	if err != nil || code != 0 {
		t.Fatalf("seed Run() code=%d err=%v", code, err)
	}

	stdout.Reset()
	code, err = Run([]string{"history", "--limit", "1"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "show disk free space") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunHistoryModeNoEntries(t *testing.T) {
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/missing.log")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code, err := Run([]string{"history", "--limit", "1"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "No audit history entries matched.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunCompletionBash(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"completion", "bash"}, strings.NewReader(""), stdout, stderr)
	if err != nil || code != 0 {
		t.Fatalf("Run() code=%d err=%v", code, err)
	}
	if !strings.Contains(stdout.String(), "cli-helper") {
		t.Fatalf("bash completion missing cli-helper: %q", stdout.String())
	}
}

func TestRunCompletionZsh(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"completion", "zsh"}, strings.NewReader(""), stdout, stderr)
	if err != nil || code != 0 {
		t.Fatalf("Run() code=%d err=%v", code, err)
	}
	if !strings.Contains(stdout.String(), "cli-helper") {
		t.Fatalf("zsh completion missing cli-helper: %q", stdout.String())
	}
}

func TestRunCompletionUnknownShell(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, _ := Run([]string{"completion", "ksh"}, strings.NewReader(""), stdout, stderr)
	if code != ExitUsage {
		t.Fatalf("expected ExitUsage for unknown shell, got %d", code)
	}
}

func TestRunQuietSuppressesWarnings(t *testing.T) {
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	// An unsupported query produces a warning; --quiet should suppress it.
	code, _ := Run([]string{"--quiet", "xyzzy frobulate"}, strings.NewReader(""), stdout, stderr)
	if code != ExitNoIntent {
		t.Fatalf("expected ExitNoIntent=%d, got %d", ExitNoIntent, code)
	}
	if strings.Contains(stdout.String(), "Warning:") {
		t.Fatalf("--quiet should suppress warnings, but got: %q", stdout.String())
	}
}

func TestRunExitCodeNoIntent(t *testing.T) {
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, _ := Run([]string{"this query matches absolutely nothing"}, strings.NewReader(""), stdout, stderr)
	if code != ExitNoIntent {
		t.Fatalf("expected ExitNoIntent=%d for unmatched query, got %d", ExitNoIntent, code)
	}
}

func TestRunExitCodeUsage(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, _ := Run([]string{}, strings.NewReader(""), stdout, stderr)
	if code != ExitUsage {
		t.Fatalf("expected ExitUsage=%d for no args, got %d", ExitUsage, code)
	}
}

func TestRunQueryTooLong(t *testing.T) {
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	longQuery := strings.Repeat("x", maxQueryBytes+1)
	code, err := Run([]string{longQuery}, strings.NewReader(""), stdout, stderr)
	if code != ExitUsage {
		t.Fatalf("expected ExitUsage for oversized query, got %d (err=%v)", code, err)
	}
}

func TestRunRunbookMode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NOSO_AUDIT_LOG_PATH", dir+"/audit.log")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code, err := Run([]string{"nginx", "is", "not", "starting"}, strings.NewReader(""), stdout, stderr)
	if err != nil || code != 0 {
		t.Fatalf("seed Run() code=%d err=%v", code, err)
	}

	stdout.Reset()
	outputPath := dir + "/runbook.md"
	code, err = Run([]string{"runbook", "--limit", "1", "--output", outputPath}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "# Runbook: nginx is not starting") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "Suggested Next Steps") {
		t.Fatalf("output file = %q", string(data))
	}
}

func TestRunLLMLogMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.jsonl")
	logger, err := llm.NewRequestLogger(path)
	if err != nil {
		t.Fatalf("NewRequestLogger() error = %v", err)
	}
	if err := logger.Append("heuristic", "heuristic-local", "why is worker 2 not up?", models.LLMInterpretResponse{
		NeedsClarification: true,
		Candidates: []models.LLMIntentCandidate{{
			Intent:     "service_troubleshoot",
			Confidence: 0.68,
		}},
	}, nil); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	t.Setenv("NOSO_LLM_LOG_PATH", path)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"llm-log", "--limit", "1"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "why is worker 2 not up?") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunLLMLogModeSince(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.jsonl")
	data := strings.Join([]string{
		`{"timestamp":"2026-04-11T03:00:00Z","provider":"heuristic","model":"heuristic-local","query":"old event","needs_clarification":false,"candidate_count":1,"top_intent":"service_status","top_confidence":0.90}`,
		`{"timestamp":"2026-04-11T05:00:00Z","provider":"heuristic","model":"heuristic-local","query":"recent event","needs_clarification":false,"candidate_count":1,"top_intent":"service_status","top_confidence":0.91}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("NOSO_LLM_LOG_PATH", path)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"llm-log", "--since", "2026-04-11T04:00:00Z"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if strings.Contains(stdout.String(), "old event") || !strings.Contains(stdout.String(), "recent event") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunLLMLogModeProviderAndErrorOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.jsonl")
	data := strings.Join([]string{
		`{"timestamp":"2026-04-11T05:00:00Z","provider":"heuristic","model":"heuristic-local","query":"healthy heuristic","needs_clarification":false,"candidate_count":1,"top_intent":"service_status","top_confidence":0.90}`,
		`{"timestamp":"2026-04-11T05:01:00Z","provider":"ollama","model":"qwen","query":"failed ollama","needs_clarification":false,"candidate_count":0,"error":"local llm timed out"}`,
		`{"timestamp":"2026-04-11T05:02:00Z","provider":"heuristic","model":"heuristic-local","query":"failed heuristic","needs_clarification":false,"candidate_count":0,"error":"local llm is unavailable"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("NOSO_LLM_LOG_PATH", path)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"llm-log", "--provider", "ollama", "--error-only"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "failed ollama") || strings.Contains(stdout.String(), "heuristic") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunLLMLogModeClarificationOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.jsonl")
	data := strings.Join([]string{
		`{"timestamp":"2026-04-11T05:00:00Z","provider":"heuristic","model":"heuristic-local","query":"plain query","needs_clarification":false,"candidate_count":1,"top_intent":"service_status","top_confidence":0.90}`,
		`{"timestamp":"2026-04-11T05:01:00Z","provider":"heuristic","model":"heuristic-local","query":"ambiguous query","needs_clarification":true,"candidate_count":2,"top_intent":"service_troubleshoot","top_confidence":0.68}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("NOSO_LLM_LOG_PATH", path)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"llm-log", "--clarification-only"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "ambiguous query") || strings.Contains(stdout.String(), "plain query") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunLLMLogModeStats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.jsonl")
	data := strings.Join([]string{
		`{"timestamp":"2026-04-11T05:00:00Z","provider":"heuristic","model":"heuristic-local","query":"plain query","needs_clarification":false,"candidate_count":1,"top_intent":"service_status","top_confidence":0.90}`,
		`{"timestamp":"2026-04-11T05:01:00Z","provider":"heuristic","model":"heuristic-local","query":"ambiguous query","needs_clarification":true,"candidate_count":2,"top_intent":"service_troubleshoot","top_confidence":0.68}`,
		`{"timestamp":"2026-04-11T05:02:00Z","provider":"ollama","model":"qwen","query":"failed ollama","needs_clarification":false,"candidate_count":0,"error":"local llm timed out"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("NOSO_LLM_LOG_PATH", path)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"llm-log", "--stats"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "LLM Log Summary") || !strings.Contains(stdout.String(), "Clarifications: 1") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunLLMLogModeCSVOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.jsonl")
	data := strings.Join([]string{
		`{"timestamp":"2026-04-11T05:01:00Z","provider":"ollama","model":"qwen","query":"failed ollama","needs_clarification":false,"candidate_count":0,"error":"local llm timed out"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("NOSO_LLM_LOG_PATH", path)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	outputPath := filepath.Join(t.TempDir(), "llm-errors.csv")
	code, err := Run([]string{"llm-log", "--provider", "ollama", "--error-only", "--format", "csv", "--output", outputPath}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "timestamp,provider,model,query") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	dataOut, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(dataOut), "failed ollama") {
		t.Fatalf("output file = %q", string(dataOut))
	}
}

func TestRunLLMLogModeStatsMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.jsonl")
	data := strings.Join([]string{
		`{"timestamp":"2026-04-11T05:00:00Z","provider":"heuristic","model":"heuristic-local","query":"ambiguous query","needs_clarification":true,"candidate_count":2,"top_intent":"service_troubleshoot","top_confidence":0.68}`,
		`{"timestamp":"2026-04-11T05:01:00Z","provider":"ollama","model":"qwen","query":"failed ollama","needs_clarification":false,"candidate_count":0,"error":"local llm timed out"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("NOSO_LLM_LOG_PATH", path)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"llm-log", "--stats", "--format", "markdown"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "## LLM Log Summary") || !strings.Contains(stdout.String(), "| Metric | Count |") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootMode(t *testing.T) {
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", filepath.Join(t.TempDir(), "troubleshoot-state.json"))
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code, err := Run([]string{"troubleshoot", "why", "is", "worker", "2", "not", "up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "Intent: troubleshoot_plan") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "systemctl status worker2 --no-pager -l") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Likely Causes:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootHistoryMode(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "why", "is", "worker", "2", "not", "up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil || code != ExitOK {
		t.Fatalf("seed Run() code=%d err=%v", code, err)
	}

	stdout.Reset()
	code, err = Run([]string{"troubleshoot-history", "--query", "why is worker 2 not up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "Summary:") || !strings.Contains(stdout.String(), "systemctl status worker2 --no-pager -l") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootResetMode(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "why", "is", "worker", "2", "not", "up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil || code != ExitOK {
		t.Fatalf("seed Run() code=%d err=%v", code, err)
	}

	stdout.Reset()
	code, err = Run([]string{"troubleshoot-reset", "--query", "why is worker 2 not up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "Cleared troubleshoot state for query") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	stdout.Reset()
	code, err = Run([]string{"troubleshoot-history", "--query", "why is worker 2 not up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "No troubleshoot history entries matched.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootClarificationUsesLatestThread(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "why", "is", "worker", "2", "not", "up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil || code != ExitOK {
		t.Fatalf("seed Run() code=%d err=%v", code, err)
	}

	stdout.Reset()
	code, err = Run([]string{"troubleshoot", "it's", "actually", "a", "pod"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe pod worker2") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "applied operator clarification") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootAdoptsSuggestedTarget(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query: "why is worker 2 not up?",
			SuggestedTargets: []troubleshoot.SuggestedTarget{{
				Family:  "kubernetes",
				Name:    "worker-2",
				Command: "kubectl describe pod worker-2",
			}},
			FamilyScores: map[string]float64{"service": -1.0, "kubernetes": 1.5},
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "check", "worker-2"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe pod worker-2") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: worker-2 (kubernetes)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "Previous discovery: No matching systemd unit name found for worker2.") {
		t.Fatalf("stdout retained stale previous discovery after target adoption: %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "Confirm the real unit name with `systemctl list-units --type=service | grep <name>` before probing systemd again.") {
		t.Fatalf("stdout retained stale service follow-up after target adoption: %q", stdout.String())
	}
}

func TestRunTroubleshootAdoptsSuggestedPVC(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query: "why is web-7c5c pending?",
			SuggestedTargets: []troubleshoot.SuggestedTarget{{
				Family:    "kubernetes-pvc",
				Name:      "web-data",
				Namespace: "prod",
				Command:   "kubectl describe pvc -n prod web-data",
			}},
			FamilyScores: map[string]float64{"kubernetes": 1.8},
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "check", "web-data"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe pvc -n prod web-data") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: web-data (kubernetes-pvc, namespace prod)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootAdoptsSuggestedDeployment(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query: "why is web failing?",
			SuggestedTargets: []troubleshoot.SuggestedTarget{{
				Family:    "kubernetes-deployment",
				Name:      "web",
				Namespace: "prod",
				Command:   "kubectl describe deployment -n prod web",
			}},
			FamilyScores: map[string]float64{"kubernetes": 1.6},
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "check", "web"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe deployment -n prod web") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: web (kubernetes-deployment, namespace prod)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootAdoptsSuggestedService(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query: "why is api unavailable?",
			SuggestedTargets: []troubleshoot.SuggestedTarget{{
				Family:    "kubernetes-service",
				Name:      "api",
				Namespace: "prod",
				Command:   "kubectl describe service -n prod api",
			}},
			FamilyScores: map[string]float64{"kubernetes": 1.5},
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "check", "api"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe service -n prod api") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: api (kubernetes-service, namespace prod)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootRefinesNamespaceForAdoptedPod(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query:        "why is worker 2 not up?",
			ActiveFamily: "kubernetes",
			ActiveTarget: "worker-2",
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "worker-2", "in", "prod"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe pod -n prod worker-2") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: worker-2 (kubernetes, namespace prod)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootRefinesNamespaceForAdoptedPVC(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query:           "why is web-7c5c pending?",
			ActiveFamily:    "kubernetes-pvc",
			ActiveTarget:    "web-data",
			ActiveNamespace: "prod",
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "web-data", "in", "prod"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe pvc -n prod web-data") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: web-data (kubernetes-pvc, namespace prod)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootRefinesNamespaceForAdoptedDeployment(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query:           "why is web failing?",
			ActiveFamily:    "kubernetes-deployment",
			ActiveTarget:    "web",
			ActiveNamespace: "prod",
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "web", "in", "prod"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe deployment -n prod web") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: web (kubernetes-deployment, namespace prod)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootRefinesNamespaceForAdoptedService(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query:           "why is api unavailable?",
			ActiveFamily:    "kubernetes-service",
			ActiveTarget:    "api",
			ActiveNamespace: "prod",
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "api", "in", "prod"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "kubectl describe service -n prod api") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: api (kubernetes-service, namespace prod)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunTroubleshootRefinesRuntimeHint(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", statePath)

	if err := troubleshoot.SaveState(statePath, troubleshoot.State{
		Threads: []troubleshoot.StateThread{{
			Query:        "why is worker 2 not up?",
			ActiveFamily: "runtime",
			ActiveTarget: "worker2-api",
		}},
	}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"troubleshoot", "it", "is", "podman"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "podman ps -a") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Adopted target: worker2-api (runtime, podman)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunUsesLLMFallbackCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok","candidates":[{"intent":"service_status","target":"worker2","tool_hint":"systemctl","confidence":0.93}]}`)
	}))
	defer server.Close()

	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_LLM_ENABLED", "true")
	t.Setenv("NOSO_LLM_ENDPOINT", server.URL)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"why", "is", "worker", "2", "not", "up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "systemctl status worker2 --no-pager -l") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunUsesLLMClarification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok","needs_clarification":true,"clarification_question":"Do you mean a systemd service, a container, or a Kubernetes pod?","candidates":[{"intent":"service_troubleshoot","target":"worker2","confidence":0.68}]}`)
	}))
	defer server.Close()

	t.Setenv("NOSO_AUDIT_LOG_PATH", t.TempDir()+"/audit.log")
	t.Setenv("NOSO_LLM_ENABLED", "true")
	t.Setenv("NOSO_LLM_ENDPOINT", server.URL)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code, err := Run([]string{"why", "is", "worker", "2", "not", "up?"}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "Intent: troubleshoot_plan") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "systemctl status worker2 --no-pager -l") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "ranked, read-only troubleshoot plan") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
