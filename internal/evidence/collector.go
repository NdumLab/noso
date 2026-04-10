package evidence

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Evidence holds what the collector learned about a command from local sources.
type Evidence struct {
	CommandName         string
	Exists              bool
	Path                string
	Kind                string
	HelpSnippet         []string
	ManAvailable        bool
	CompletionPath      string
	VerificationSources []string
}

// Collector runs local probes with a bounded timeout.
type Collector struct {
	timeout time.Duration
}

func NewCollector() Collector {
	return Collector{timeout: 2 * time.Second}
}

// Lookup probes the local system for information about name.
// name must come from trusted/internal code, not raw user input.
func (c Collector) Lookup(name string) Evidence {
	ev := Evidence{CommandName: name}

	// exec.LookPath replaces bash -c "command -v name" — no shell involved.
	if path, err := exec.LookPath(name); err == nil {
		ev.Exists = true
		ev.Path = path
		ev.Kind = "file"
		ev.VerificationSources = append(ev.VerificationSources, "exec.LookPath")
	} else if isKnownBuiltin(name) {
		ev.Exists = true
		ev.Kind = "builtin"
		ev.VerificationSources = append(ev.VerificationSources, "builtin")
	}

	if !ev.Exists {
		return ev
	}

	// Short help snippet for confidence scoring and flag verification.
	if snippet := c.helpSnippet(name, ev.Kind, ev.Path); len(snippet) > 0 {
		ev.HelpSnippet = snippet
		ev.VerificationSources = append(ev.VerificationSources, name+" --help")
	}

	// man -w just prints the path — fast, no rendering.
	if c.outputContainsSubstr("man", []string{"-w", name}, "/") {
		ev.ManAvailable = true
		ev.VerificationSources = append(ev.VerificationSources, "man")
	}

	// Shell completion script.
	for _, dir := range []string{
		"/usr/share/bash-completion/completions",
		"/etc/bash_completion.d",
	} {
		p := dir + "/" + name
		if _, err := os.Stat(p); err == nil {
			ev.CompletionPath = p
			ev.VerificationSources = append(ev.VerificationSources, "completion-script")
			break
		}
	}

	return ev
}

func (c Collector) helpSnippet(name, kind, path string) []string {
	if kind == "builtin" {
		// Shell builtins require bash; name is from our hardcoded list.
		if lines := c.runDirect("bash", []string{"-c", "help " + name}); len(lines) > 0 {
			return firstN(lines, 3)
		}
		return nil
	}
	for _, flag := range []string{"--help", "-h"} {
		if lines := c.runDirect(path, []string{flag}); len(lines) > 0 {
			return firstN(lines, 3)
		}
	}
	return nil
}

// outputContainsSubstr runs cmd with args and returns true if output contains substr.
func (c Collector) outputContainsSubstr(cmd string, args []string, substr string) bool {
	for _, l := range c.runDirect(cmd, args) {
		if strings.Contains(l, substr) {
			return true
		}
	}
	return false
}

// runDirect executes a command directly without a shell intermediary.
// Many --help invocations exit non-zero; we still capture their output.
func (c Collector) runDirect(name string, args []string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run()

	var lines []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if t := strings.TrimSpace(line); t != "" {
			lines = append(lines, t)
		}
	}
	return lines
}

// RunLinesForDetection executes a fully-hardcoded shell script (e.g.
// "kubectl config current-context") and returns its output lines.
// Never pass user-supplied input to this function.
func (c Collector) RunLinesForDetection(script string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", script)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return nil
	}

	var lines []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if t := strings.TrimSpace(line); t != "" {
			lines = append(lines, t)
		}
	}
	return lines
}

// isKnownBuiltin returns true for bash shell builtins noso may document.
func isKnownBuiltin(name string) bool {
	switch name {
	case "cd", "echo", "test", "read", "export", "source", "alias",
		"unalias", "pwd", "set", "unset", "trap", "type", "help",
		"true", "false", "exit", "return", "break", "continue":
		return true
	}
	return false
}

func firstN(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[:n]
}
