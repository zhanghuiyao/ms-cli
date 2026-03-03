package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadTool(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &ReadTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": tmpFile,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Output)
	}
	if result.Output != content {
		t.Errorf("expected %q, got %q", content, result.Output)
	}
}

func TestReadToolNotFound(t *testing.T) {
	tool := &ReadTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "/nonexistent/file.txt",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for nonexistent file")
	}
}

func TestReadToolPathTraversal(t *testing.T) {
	tool := &ReadTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "../../../etc/passwd",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for path traversal attempt")
	}
}

func TestWriteTool(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	tool := &WriteTool{}
	content := "Test content"
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":    tmpFile,
		"content": content,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Output)
	}

	// Verify file was written
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}
}

func TestEditTool(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	original := "Hello, World!"
	if err := os.WriteFile(tmpFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &EditTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"path":     tmpFile,
		"old_text": "World",
		"new_text": "Go",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Output)
	}

	// Verify edit
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Hello, Go!"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestGlobTool(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte(""), 0644)

	tool := &GlobTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": filepath.Join(tmpDir, "*.go"),
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Output)
	}

	// Should find 2 .go files
	data, _ := result.Data["count"].(int)
	if data != 2 {
		t.Errorf("expected 2 files, got %d", data)
	}
}
