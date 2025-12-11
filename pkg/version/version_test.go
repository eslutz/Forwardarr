package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	origDate := Date

	// Restore original values at the end
	defer func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	}()

	// Test with custom values
	Version = "1.2.3"
	Commit = "abc123"
	Date = "2024-01-01"

	result := String()
	expected := "Forwardarr 1.2.3 (commit: abc123, built: 2024-01-01)"

	if result != expected {
		t.Errorf("String() = %q, want %q", result, expected)
	}

	// Test with dev values
	Version = "dev"
	Commit = "unknown"
	Date = "unknown"

	result = String()
	expected = "Forwardarr dev (commit: unknown, built: unknown)"

	if result != expected {
		t.Errorf("String() = %q, want %q", result, expected)
	}
}

func TestVersionVariables(t *testing.T) {
	// Test that default values are set
	if Version == "" {
		t.Error("Version should have a default value")
	}
	if Commit == "" {
		t.Error("Commit should have a default value")
	}
	if Date == "" {
		t.Error("Date should have a default value")
	}
}

func TestStringContainsForwardarr(t *testing.T) {
	result := String()
	if !strings.Contains(result, "Forwardarr") {
		t.Errorf("String() = %q, should contain 'Forwardarr'", result)
	}
}
