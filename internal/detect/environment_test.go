package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
)

func TestDistroFamily(t *testing.T) {
	cases := []struct {
		id     string
		idLike string
		want   string
	}{
		{"rhel", "", "rhel"},
		{"centos", "", "rhel"},
		{"rocky", "", "rhel"},
		{"almalinux", "", "rhel"},
		{"fedora", "", "fedora"},
		{"ubuntu", "", "debian"},
		{"debian", "", "debian"},
		{"linuxmint", "ubuntu debian", "debian"},
		{"pop", "ubuntu debian", "debian"},
		{"opensuse", "", "suse"},
		{"sles", "", "suse"},
		{"arch", "", "arch"},
		{"manjaro", "arch", "arch"},
		{"alpine", "", "unknown"},
		{"", "", "unknown"},
		// ID_LIKE fallback
		{"mylinux", "rhel fedora", "rhel"},
		{"mylinux", "ubuntu", "debian"},
	}
	for _, tc := range cases {
		got := distroFamily(tc.id, tc.idLike)
		if got != tc.want {
			t.Errorf("distroFamily(%q, %q) = %q, want %q", tc.id, tc.idLike, got, tc.want)
		}
	}
}

func TestParseOSRelease(t *testing.T) {
	content := `ID=rhel
VERSION_ID="9.7"
PRETTY_NAME="Red Hat Enterprise Linux 9.7 (Plow)"
ID_LIKE="fedora"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "os-release")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	vals, err := parseOSRelease(path)
	if err != nil {
		t.Fatalf("parseOSRelease() error = %v", err)
	}
	if vals["ID"] != "rhel" {
		t.Errorf("ID = %q, want rhel", vals["ID"])
	}
	if vals["VERSION_ID"] != "9.7" {
		t.Errorf("VERSION_ID = %q, want 9.7 (quotes stripped)", vals["VERSION_ID"])
	}
	if vals["ID_LIKE"] != "fedora" {
		t.Errorf("ID_LIKE = %q, want fedora", vals["ID_LIKE"])
	}
}

func TestParseOSReleaseMissingFile(t *testing.T) {
	_, err := parseOSRelease("/does/not/exist/os-release")
	if err == nil {
		t.Fatal("parseOSRelease() expected error for missing file")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testfile")
	if fileExists(path) {
		t.Error("fileExists() = true for non-existent file")
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !fileExists(path) {
		t.Error("fileExists() = false for existing file")
	}
	// Directories are not files.
	if fileExists(dir) {
		t.Error("fileExists() = true for a directory")
	}
}

func TestKubeConfigPathRespectsEnv(t *testing.T) {
	dir := t.TempDir()
	fakeCfg := filepath.Join(dir, "config")
	if err := os.WriteFile(fakeCfg, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("KUBECONFIG", fakeCfg)
	if got := kubeConfigPath(); got != fakeCfg {
		t.Errorf("kubeConfigPath() = %q, want %q", got, fakeCfg)
	}
}

func TestKubeConfigPathNonExistentEnv(t *testing.T) {
	t.Setenv("KUBECONFIG", "/does/not/exist/kubeconfig")
	if got := kubeConfigPath(); got != "" {
		t.Errorf("kubeConfigPath() = %q, want empty for non-existent KUBECONFIG", got)
	}
}

func TestKubeContextNoKubectl(t *testing.T) {
	// On a host without kubectl the function should return "".
	// We use a fresh collector; if kubectl isn't installed this always returns "".
	// The test is a no-panic/correctness check that always passes.
	collector := evidence.NewCollector()
	ev := collector.Lookup("kubectl")
	if ev.Exists {
		t.Skip("kubectl is installed; skipping no-kubectl test")
	}
	got := kubeContext(collector)
	if got != "" {
		t.Errorf("kubeContext() = %q, want empty when kubectl is absent", got)
	}
}

func TestCollectorCommandLinesEcho(t *testing.T) {
	lines := collectorCommandLines("echo noso_detect_test")
	found := false
	for _, l := range lines {
		if l == "noso_detect_test" {
			found = true
		}
	}
	if !found {
		t.Errorf("collectorCommandLines(echo) = %v, expected noso_detect_test", lines)
	}
}

func TestLocalReturnsNonEmpty(t *testing.T) {
	env, err := Local()
	if err != nil {
		t.Fatalf("Local() error = %v", err)
	}
	if env.Distro == "" {
		t.Error("Local().Distro should not be empty on a Linux host")
	}
	if env.PackageManager == "" {
		t.Error("Local().PackageManager should not be empty on a Linux host")
	}
}

func TestDetectPackageManager(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		{"rhel", "dnf"},
		{"centos", "dnf"},
		{"fedora", "dnf"},
		{"ubuntu", "apt"},
		{"debian", "apt"},
		{"opensuse", "zypper"},
		{"arch", "pacman"},
		{"alpine", "unknown"},
	}
	for _, tc := range cases {
		got := detectPackageManager(tc.id, "")
		if got != tc.want {
			t.Errorf("detectPackageManager(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}
