package buildinfo

import "testing"

func TestStringIncludesVersion(t *testing.T) {
	value := String()
	if value == "" {
		t.Fatal("String() should not be empty")
	}
	if value[:8] != "version=" {
		t.Fatalf("String() = %q", value)
	}
}
