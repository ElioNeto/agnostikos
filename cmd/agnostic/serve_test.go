package agnostic

import (
	"testing"
)

// resetServeFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetServeFlags() {
	servePort = "8080"
	serveOpen = false
}

func TestServeCmd_Use(t *testing.T) {
	resetServeFlags()
	if serveCmd.Use != "serve" {
		t.Errorf("expected Use 'serve', got %q", serveCmd.Use)
	}
	if serveCmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

func TestServeCmd_PortFlagDefault(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("port")
	if f == nil {
		t.Fatal("expected --port flag to be defined")
	}
	if f.DefValue != "8080" {
		t.Errorf("expected default port '8080', got %q", f.DefValue)
	}
}

func TestServeCmd_PortFlagShort(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("port")
	if f == nil {
		t.Fatal("expected --port flag to be defined")
	}
	if f.Shorthand != "p" {
		t.Errorf("expected shorthand 'p', got %q", f.Shorthand)
	}
}

func TestServeCmd_OpenFlagDefault(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("open")
	if f == nil {
		t.Fatal("expected --open flag to be defined")
	}
	if f.DefValue != "false" {
		t.Errorf("expected default open 'false', got %q", f.DefValue)
	}
}

func TestServeCmd_OpenFlagShort(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("open")
	if f == nil {
		t.Fatal("expected --open flag to be defined")
	}
	if f.Shorthand != "o" {
		t.Errorf("expected shorthand 'o', got %q", f.Shorthand)
	}
}

func TestServeCmd_PortFlagType(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("port")
	if f == nil {
		t.Fatal("expected --port flag to be defined")
	}
	if f.Value.Type() != "string" {
		t.Errorf("expected port flag type 'string', got %q", f.Value.Type())
	}
}
