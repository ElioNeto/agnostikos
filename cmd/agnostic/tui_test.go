package agnostic

import (
	"testing"
)

func TestTUICmd_Use(t *testing.T) {
	if tuiCmd.Use != "tui" {
		t.Errorf("expected Use 'tui', got %q", tuiCmd.Use)
	}
	if tuiCmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

func TestTUICmd_NoArgsValidation(t *testing.T) {
	// Directly test cobra.NoArgs validation without going through ExecuteC,
	// since cobra's Find cannot route unknown args to the tui command handler.
	err := tuiCmd.Args(tuiCmd, []string{"extra-arg"})
	if err == nil {
		t.Fatal("expected error for extra args, got nil")
	}
	// cobra.NoArgs returns a non-nil error for extra arguments
	// (the error text varies by cobra version — just check it's non-nil)
}

func TestTUICmd_NoArgsEmpty(t *testing.T) {
	err := tuiCmd.Args(tuiCmd, []string{})
	if err != nil {
		t.Errorf("expected no error for empty args, got: %v", err)
	}
}

func TestTUICmd_CanBeCalled(t *testing.T) {
	// Verify the command is registered under rootCmd
	cmd, _, err := rootCmd.Find([]string{"tui"})
	if err != nil {
		t.Fatalf("expected tui command to be findable, got: %v", err)
	}
	if cmd == nil || cmd.Name() != "tui" {
		t.Errorf("expected to find tui command, got: %v", cmd)
	}
}

func TestTUICmd_ArgsAnnotation(t *testing.T) {
	if tuiCmd.Args == nil {
		t.Fatal("expected Args to be set")
	}
}
