package agnostic

import (
	"strings"
	"testing"
)

// resetDotfilesFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetDotfilesFlags() {
	dotfilesFrom = ""
	dotfilesForce = false
}

func TestDotfilesCmd_Use(t *testing.T) {
	resetDotfilesFlags()
	if dotfilesCmd.Use != "dotfiles" {
		t.Errorf("expected Use 'dotfiles', got %q", dotfilesCmd.Use)
	}
	if dotfilesCmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

func TestDotfilesCmd_SubcommandsRegistered(t *testing.T) {
	resetDotfilesFlags()
	subs := dotfilesCmd.Commands()
	names := make(map[string]bool, len(subs))
	for _, sub := range subs {
		names[sub.Name()] = true
	}

	if !names["apply"] {
		t.Error("expected 'apply' subcommand to be registered")
	}
	if !names["list"] {
		t.Error("expected 'list' subcommand to be registered")
	}
	if !names["diff"] {
		t.Error("expected 'diff' subcommand to be registered")
	}
}

func TestDotfilesCmd_ApplySubcommandExists(t *testing.T) {
	resetDotfilesFlags()
	cmd, _, err := dotfilesCmd.Find([]string{"apply"})
	if err != nil {
		t.Fatalf("expected apply subcommand to be findable, got: %v", err)
	}
	if cmd == nil || cmd.Name() != "apply" {
		t.Errorf("expected to find apply subcommand, got: %v", cmd)
	}
}

func TestDotfilesCmd_ListSubcommandExists(t *testing.T) {
	resetDotfilesFlags()
	cmd, _, err := dotfilesCmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("expected list subcommand to be findable, got: %v", err)
	}
	if cmd == nil || cmd.Name() != "list" {
		t.Errorf("expected to find list subcommand, got: %v", cmd)
	}
}

func TestDotfilesCmd_DiffSubcommandExists(t *testing.T) {
	resetDotfilesFlags()
	cmd, _, err := dotfilesCmd.Find([]string{"diff"})
	if err != nil {
		t.Fatalf("expected diff subcommand to be findable, got: %v", err)
	}
	if cmd == nil || cmd.Name() != "diff" {
		t.Errorf("expected to find diff subcommand, got: %v", cmd)
	}
}

func TestDotfilesCmd_ApplyFromFlag(t *testing.T) {
	resetDotfilesFlags()
	f := dotfilesApplyCmd.Flags().Lookup("from")
	if f == nil {
		t.Fatal("expected --from flag on apply to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default from '', got %q", f.DefValue)
	}
}

func TestDotfilesCmd_ApplyForceFlag(t *testing.T) {
	resetDotfilesFlags()
	f := dotfilesApplyCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("expected --force flag on apply to be defined")
	}
	if f.DefValue != "false" {
		t.Errorf("expected default force 'false', got %q", f.DefValue)
	}
}

func TestDotfilesCmd_ApplyForceFlagShort(t *testing.T) {
	resetDotfilesFlags()
	f := dotfilesApplyCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("expected --force flag on apply to be defined")
	}
	if f.Shorthand != "f" {
		t.Errorf("expected shorthand 'f', got %q", f.Shorthand)
	}
}

func TestDotfilesCmd_HelpContainsApply(t *testing.T) {
	resetDotfilesFlags()
	if !strings.Contains(dotfilesCmd.Long, "apply") {
		t.Error("expected help text to mention 'apply'")
	}
	if !strings.Contains(dotfilesCmd.Long, "list") {
		t.Error("expected help text to mention 'list'")
	}
	if !strings.Contains(dotfilesCmd.Long, "diff") {
		t.Error("expected help text to mention 'diff'")
	}
}
