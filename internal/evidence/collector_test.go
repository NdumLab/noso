package evidence

import "testing"

func TestIsKnownBuiltin(t *testing.T) {
	builtins := []string{"cd", "echo", "test", "read", "export", "pwd", "exit"}
	for _, name := range builtins {
		if !isKnownBuiltin(name) {
			t.Errorf("isKnownBuiltin(%q) = false, want true", name)
		}
	}
	nonBuiltins := []string{"ls", "grep", "awk", "kubectl", "git", "df", ""}
	for _, name := range nonBuiltins {
		if isKnownBuiltin(name) {
			t.Errorf("isKnownBuiltin(%q) = true, want false", name)
		}
	}
}

func TestFirstN(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	if got := firstN(lines, 3); len(got) != 3 {
		t.Errorf("firstN(5 lines, 3) = %d lines, want 3", len(got))
	}
	if got := firstN(lines, 10); len(got) != 5 {
		t.Errorf("firstN(5 lines, 10) = %d lines, want 5", len(got))
	}
	if got := firstN(nil, 3); len(got) != 0 {
		t.Errorf("firstN(nil, 3) = %d lines, want 0", len(got))
	}
}

func TestLookupMissingCommand(t *testing.T) {
	c := NewCollector()
	ev := c.Lookup("__noso_no_such_command_xyz__")
	if ev.Exists {
		t.Error("Lookup of non-existent command should not report Exists=true")
	}
	if ev.Path != "" {
		t.Errorf("Path should be empty for missing command, got %q", ev.Path)
	}
	if len(ev.VerificationSources) != 0 {
		t.Errorf("VerificationSources should be empty for missing command, got %v", ev.VerificationSources)
	}
}

func TestLookupKnownCommand(t *testing.T) {
	c := NewCollector()
	// 'ls' is available on every Linux system we target.
	ev := c.Lookup("ls")
	if !ev.Exists {
		t.Skip("ls not found — skipping test in this environment")
	}
	if ev.Path == "" {
		t.Error("Path should not be empty for a found command")
	}
	if ev.Kind != "file" {
		t.Errorf("Kind = %q, want file", ev.Kind)
	}
}

func TestLookupBuiltinCommand(t *testing.T) {
	c := NewCollector()
	// "source" is a bash builtin that rarely ships as a standalone binary.
	// If the host has /usr/bin/source, the Kind will be "file" — acceptable.
	// Either way it must report Exists=true.
	ev := c.Lookup("source")
	if !ev.Exists {
		t.Error("Lookup(source) should report Exists=true (builtin or file)")
	}
	if ev.Kind != "builtin" && ev.Kind != "file" {
		t.Errorf("Kind = %q, want builtin or file", ev.Kind)
	}
}

func TestRunLinesForDetection(t *testing.T) {
	c := NewCollector()
	lines := c.RunLinesForDetection("echo hello_noso_test")
	found := false
	for _, l := range lines {
		if l == "hello_noso_test" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("RunLinesForDetection(echo) = %v, want line containing hello_noso_test", lines)
	}
}

func TestRunLinesForDetectionFailingCommand(t *testing.T) {
	c := NewCollector()
	lines := c.RunLinesForDetection("__noso_no_such_cmd_xyz__ 2>/dev/null")
	// A failing command should return nil, not panic.
	_ = lines
}

func TestOutputContainsSubstrFalse(t *testing.T) {
	c := NewCollector()
	// 'echo' output will not contain this unlikely string.
	got := c.outputContainsSubstr("echo", []string{"noso_test_output"}, "xyz_no_match_sentinel")
	if got {
		t.Error("outputContainsSubstr should return false when substr is absent")
	}
}

func TestOutputContainsSubstrTrue(t *testing.T) {
	c := NewCollector()
	got := c.outputContainsSubstr("echo", []string{"found_it"}, "found_it")
	if !got {
		t.Error("outputContainsSubstr should return true when substr is present")
	}
}
