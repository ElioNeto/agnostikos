package agnostic

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestInstallCmd(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		expectedErr error
	}{
		{
			name:        "resource name required",
			args:        []string{},
			expectedErr: fmt.Errorf("resource name is required"),
		},
		{
			name:        "create resource directory success",
			args:        []string{"test-resource"},
			expectedErr: nil,
		},
		{
			name:        "create resource directory failure",
			args:        []string{"test-resource"},
			expectedErr: errors.New("failed to create resource directory"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock os.MkdirAll
			mockMkdirAll := func(path string, perm os.FileMode) error {
				if tc.expectedErr != nil && path == filepath.Join("/path/to/resources", "test-resource") {
					return tc.expectedErr
				}
				return nil
			}

			originalMkdirAll := os.MkdirAll
			os.MkdirAll = mockMkdirAll
			defer func() { os.MkdirAll = originalMkdirAll }()

			err := installCmd.RunE(context.Background(), tc.args)
			if err != tc.expectedErr {
				t.Errorf("Expected error: %v, got: %v", tc.expectedErr, err)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	testCases := []struct {
		name        string
		expectedErr error
	}{
		{
			name:        "execute install command success",
			expectedErr: nil,
		},
		{
			name:        "execute install command failure",
			expectedErr: errors.New("failed to create resource directory"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock os.MkdirAll
			mockMkdirAll := func(path string, perm os.FileMode) error {
				if tc.expectedErr != nil && path == filepath.Join("/path/to/resources", "test-resource") {
					return tc.expectedErr
				}
				return nil
			}

			originalMkdirAll := os.MkdirAll
			os.MkdirAll = mockMkdirAll
			defer func() { os.MkdirAll = originalMkdirAll }()

			err := Execute()
			if err != tc.expectedErr {
				t.Errorf("Expected error: %v, got: %v", tc.expectedErr, err)
			}
		})
	}
}