package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

// FuzzResolve verifies that Resolve never panics, always returns a
// non-empty IntentID, and always returns a recognised Risk level for any
// query string an operator might type or paste.
//
// The collector performs real local probes (exec.LookPath, --help).  These
// are fast for the seed corpus used in CI but make sustained mutation
// fuzzing slow.  Full mutation fuzzing is intended for development only.
//
// Run seed corpus only (fast, used in CI):
//
//	go test -run=FuzzResolve ./internal/registry/
//
// Run full mutation fuzzing (development only):
//
//	go test -fuzz=FuzzResolve -fuzztime=60s ./internal/registry/
func FuzzResolve(f *testing.F) {
	seeds := []string{
		// Normal queries
		"show disk free space",
		"git log",
		"git diff",
		"what process is on port 8080",
		"show network interfaces",
		"list docker containers",
		"show pods in namespace prod",
		"list helm releases",
		"explain terraform destroy",
		"explain rm -rf /",
		// Troubleshoot paths
		"pod is in CrashLoopBackOff",
		"nginx is not starting",
		"connection refused on port 5432",
		"no space left on device",
		// Explain mode
		"explain git reset --hard HEAD~1",
		"what does kubectl delete do",
		// Adversarial / injection attempts
		"",
		"; rm -rf /",
		"$(echo injected)",
		"`id`",
		"show disk free space; rm -rf /",
		"explain `rm -rf /`",
		// Whitespace and padding
		"   ",
		"\t\n",
		strings.Repeat("show ", 500),
		// Non-ASCII
		"показать дисковое пространство",
		"磁盘使用情况",
		// Very long input
		strings.Repeat("a", 65536),
		// Null bytes
		"show\x00disk\x00free",
		"\x00",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, query string) {
		// Reject inputs that exceed the CLI's own hard limit — the CLI
		// enforces this before calling Resolve, so we skip rather than
		// testing an input the real binary would have already rejected.
		if len(query) > 64*1024 {
			t.Skip()
		}

		resp, err := Resolve(query, models.Environment{}, evidence.NewCollector())
		if err != nil {
			t.Errorf("Resolve(%q) returned unexpected error: %v", query, err)
			return
		}
		if resp.IntentID == "" {
			t.Errorf("Resolve(%q) returned empty IntentID", query)
		}
		switch resp.Risk {
		case "Low", "Medium", "High", "":
			// valid
		default:
			t.Errorf("Resolve(%q) returned unexpected Risk %q", query, resp.Risk)
		}
	})
}
