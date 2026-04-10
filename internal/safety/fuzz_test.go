package safety

import (
	"strings"
	"testing"
)

// FuzzClassify verifies that Classify never panics and always returns one
// of the three known risk levels for any input string.
//
// Run seed corpus only (fast, used in CI):
//
//	go test -run=FuzzClassify ./internal/safety/
//
// Run full mutation fuzzing (development only):
//
//	go test -fuzz=FuzzClassify -fuzztime=60s ./internal/safety/
func FuzzClassify(f *testing.F) {
	seeds := []string{
		// Ordinary read-only commands
		"", "ls", "df -h", "free -h", "ps aux", "git status",
		// High-risk patterns
		"rm -rf /",
		"rm -r /tmp/foo",
		"rm -f /etc/hosts",
		"git reset --hard HEAD~1",
		"git clean -f",
		"git push --force origin main",
		"docker system prune",
		"podman system prune",
		"kubectl delete pod mypod",
		"terraform destroy",
		"terraform apply",
		"ansible-playbook site.yml",
		"drop table users",
		"truncate sessions",
		"dd if=/dev/zero of=/dev/sda",
		// Safe overrides that contain high-risk substrings
		"terraform plan",
		"ansible-playbook --check site.yml",
		"ansible-playbook --syntax-check site.yml",
		"helm template mychart",
		// Medium-risk patterns
		"systemctl restart nginx",
		"chmod 755 /var/www",
		"chown root:root /etc/passwd",
		"helm install myapp ./chart",
		// Injection-style adversarial inputs
		"; rm -rf /",
		"$(kubectl delete pod foo)",
		"`rm -rf /`",
		"rm -rf / --no-preserve-root",
		"rm -rf /; echo pwned",
		// Whitespace variants
		"  rm -rf /  ",
		"\trm -rf /\n",
		strings.Repeat("rm -rf ", 200),
		// Non-ASCII and unusual bytes
		"rm -rf /café",
		"\x00\x01\x02",
		"rm\x00-rf",
		// Very long input
		strings.Repeat("a", 65536),
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, cmd string) {
		result := Classify(cmd)
		switch result {
		case RiskLow, RiskMedium, RiskHigh:
			// all valid — no action needed
		default:
			t.Errorf("Classify(%q) = %q: not one of Low/Medium/High", cmd, result)
		}
	})
}
