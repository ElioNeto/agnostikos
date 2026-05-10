package agnostic

import (
	"testing"
)

// resetServeFlags resets package-level variables to their defaults
// so that tests don't leak state between each other.
func resetServeFlags() {
	serveListen = "127.0.0.1:8080"
	serveOpen = false
	serveToken = ""
	serveTLSCert = ""
	serveTLSKey = ""
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

func TestServeCmd_ListenFlagDefault(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("listen")
	if f == nil {
		t.Fatal("expected --listen flag to be defined")
	}
	if f.DefValue != "127.0.0.1:8080" {
		t.Errorf("expected default listen '127.0.0.1:8080', got %q", f.DefValue)
	}
}

func TestServeCmd_ListenFlagShort(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("listen")
	if f == nil {
		t.Fatal("expected --listen flag to be defined")
	}
	if f.Shorthand != "l" {
		t.Errorf("expected shorthand 'l', got %q", f.Shorthand)
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

func TestServeCmd_TokenFlag(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("token")
	if f == nil {
		t.Fatal("expected --token flag to be defined")
	}
	if f.DefValue != "" {
		t.Errorf("expected default token '', got %q", f.DefValue)
	}
}

func TestServeCmd_TLSCertFlag(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("tls-cert")
	if f == nil {
		t.Fatal("expected --tls-cert flag to be defined")
	}
}

func TestServeCmd_TLSKeyFlag(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("tls-key")
	if f == nil {
		t.Fatal("expected --tls-key flag to be defined")
	}
}

func TestServeCmd_ListenFlagType(t *testing.T) {
	resetServeFlags()
	f := serveCmd.Flags().Lookup("listen")
	if f == nil {
		t.Fatal("expected --listen flag to be defined")
	}
	if f.Value.Type() != "string" {
		t.Errorf("expected listen flag type 'string', got %q", f.Value.Type())
	}
}
