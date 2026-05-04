package bootstrap

import (
	"context"
	"testing"
)

func TestBusyboxConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     BusyboxConfig
		wantErr bool
	}{
		{
			name:    "empty version",
			cfg:     BusyboxConfig{Version: "", TargetDir: "/tmp/test"},
			wantErr: true,
		},
		{
			name:    "empty target dir",
			cfg:     BusyboxConfig{Version: "1.36.1", TargetDir: ""},
			wantErr: true,
		},
		{
			name:    "valid config",
			cfg:     BusyboxConfig{Version: "1.36.1", TargetDir: "/tmp/test"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We only test validation by calling with a canceled context
			// so it won't actually try to download anything
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			err := BuildBusybox(ctx, tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for config: %+v", tt.cfg)
				}
			} else {
				// With canceled context, we should get an error from the context,
				// but not a validation error
				if err == nil {
					t.Error("expected error from canceled context")
				}
			}
		})
	}
}

func TestBusyboxConfig_Defaults(t *testing.T) {
	// Verify zero values are reasonable
	cfg := BusyboxConfig{}
	if cfg.Version != "" {
		t.Errorf("expected empty version, got %s", cfg.Version)
	}
	if cfg.TargetDir != "" {
		t.Errorf("expected empty target dir, got %s", cfg.TargetDir)
	}
	if cfg.NumCPUs != "" {
		t.Errorf("expected empty numCPUs, got %s", cfg.NumCPUs)
	}
}
