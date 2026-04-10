package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
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
