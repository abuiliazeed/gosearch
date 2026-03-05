package cli

import "testing"

func TestRootCommandIncludesTUI(t *testing.T) {
	t.Helper()

	found := false
	for _, command := range rootCmd.Commands() {
		if command.Name() == "tui" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected root command list to include \"tui\"")
	}
}

func TestRootCommandIncludesMDExport(t *testing.T) {
	t.Helper()

	found := false
	for _, command := range rootCmd.Commands() {
		if command.Name() == "md-export" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected root command list to include \"md-export\"")
	}
}
