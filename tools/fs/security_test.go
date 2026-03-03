package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSecurePath_ValidPaths(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		baseDir string
		wantErr bool
	}{
		{
			name:    "simple file in base",
			path:    "test.txt",
			baseDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "nested file",
			path:    "subdir/test.txt",
			baseDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "absolute path within base",
			path:    filepath.Join(tmpDir, "test.txt"),
			baseDir: tmpDir,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SecurePath(tt.path, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got == "" {
				t.Error("SecurePath() returned empty path without error")
			}
		})
	}
}

func TestSecurePath_TraversalAttacks(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		baseDir string
		wantErr bool
	}{
		{
			name:    "simple traversal",
			path:    "../../../etc/passwd",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "traversal in middle",
			path:    "foo/../../../etc/passwd",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "double dots encoded",
			path:    "..%2f..%2fetc/passwd",
			baseDir: tmpDir,
			wantErr: false, // URL encoding should be handled by caller
		},
		{
			name:    "traversal with valid prefix",
			path:    "valid/../../../../etc/passwd",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "dot dot slash variations",
			path:    "..\\..\\windows\\system32\\config\\sam",
			baseDir: tmpDir,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SecurePath(tt.path, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurePath_SymlinkAttacks(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping symlink tests when running as root")
	}

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Create a file outside the subdirectory
	outsideFile := filepath.Join(tmpDir, "secret.txt")
	os.WriteFile(outsideFile, []byte("secret"), 0644)

	// Create a symlink inside subdir pointing outside
	symlinkPath := filepath.Join(subDir, "link")
	os.Symlink(outsideFile, symlinkPath)

	// This should be detected as a traversal attack
	_, err := SecurePath(symlinkPath, subDir)
	if err == nil {
		t.Error("SecurePath() should detect symlink pointing outside base directory")
	}
}

func TestReadTool_Security(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file inside tmpDir
	safeFile := filepath.Join(tmpDir, "safe.txt")
	os.WriteFile(safeFile, []byte("safe content"), 0644)

	// Create a file outside tmpDir
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	os.WriteFile(outsideFile, []byte("secret"), 0644)

	tool := &ReadTool{WorkDir: tmpDir}

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "safe file access",
			path:      "safe.txt",
			wantError: false,
		},
		{
			name:      "traversal attack",
			path:      "../" + filepath.Base(outsideDir) + "/secret.txt",
			wantError: true,
		},
		{
			name:      "absolute path outside base",
			path:      outsideFile,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), map[string]any{
				"path": tt.path,
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError && result.Success {
				t.Errorf("expected security error for path %s, but succeeded", tt.path)
			}
			if !tt.wantError && !result.Success {
				t.Errorf("expected success for path %s, but got error: %v", tt.path, result.Error)
			}
		})
	}
}

func TestWriteTool_Security(t *testing.T) {
	tmpDir := t.TempDir()
	tool := &WriteTool{WorkDir: tmpDir}

	// Attempt to write outside the base directory
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":    "../../etc/passwd",
		"content": "malicious",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("WriteTool should block directory traversal attempts")
	}
}

func TestEditTool_Security(t *testing.T) {
	tmpDir := t.TempDir()
	tool := &EditTool{WorkDir: tmpDir}

	// Attempt to edit outside the base directory
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     "../../etc/passwd",
		"old_text": "root",
		"new_text": "hacked",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("EditTool should block directory traversal attempts")
	}
}
