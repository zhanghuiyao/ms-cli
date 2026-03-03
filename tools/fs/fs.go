package fs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vigo999/ms-cli/tools"
)

// ReadTool implements file reading functionality.
type ReadTool struct {
	WorkDir string // Base directory for path validation
}

func (t *ReadTool) Name() string        { return "fs_read" }
func (t *ReadTool) Description() string { return "Read the contents of a file" }

func (t *ReadTool) Execute(ctx context.Context, params map[string]any) (tools.Result, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return tools.Result{Success: false, Output: fmt.Sprintf("path parameter is required")}, nil
	}

	// Security: prevent directory traversal
	baseDir := t.WorkDir
	if baseDir == "" {
		baseDir = GetWorkDir()
	}
	
	securePath, err := SecurePath(path, baseDir)
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("security check failed: %v", err)}, nil
	}

	data, err := os.ReadFile(securePath)
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("read file: %v", err)}, nil
	}

	return tools.Result{
		Success: true,
		Output:  string(data),
		Data: map[string]any{
			"path":     securePath,
			"size":     len(data),
			"lines":    strings.Count(string(data), "\n") + 1,
		},
	}, nil
}

// ReadDefinition returns the schema for fs_read.
func ReadDefinition() tools.Definition {
	return tools.Definition{
		Name:        "fs_read",
		Description: "Read the contents of a file. Supports text files and code files.",
		Parameters: tools.Parameters{
			Type: "object",
			Properties: map[string]tools.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to read",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of lines to read (optional)",
				},
				"offset": {
					Type:        "integer",
					Description: "Line number to start reading from (1-indexed, optional)",
				},
			},
			Required: []string{"path"},
		},
	}
}

// WriteTool implements file writing functionality.
type WriteTool struct {
	WorkDir string // Base directory for path validation
}

func (t *WriteTool) Name() string        { return "fs_write" }
func (t *WriteTool) Description() string { return "Write content to a file" }

func (t *WriteTool) Execute(ctx context.Context, params map[string]any) (tools.Result, error) {
	path, _ := params["path"].(string)
	content, _ := params["content"].(string)

	if path == "" {
		return tools.Result{Success: false, Output: fmt.Sprintf("path parameter is required")}, nil
	}

	// Security check
	baseDir := t.WorkDir
	if baseDir == "" {
		baseDir = GetWorkDir()
	}

	securePath, err := SecurePath(path, baseDir)
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("security check failed: %v", err)}, nil
	}

	// Create parent directories if needed
	dir := filepath.Dir(securePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("create directory: %v", err)}, nil
	}

	// Check if file exists for diff
	var oldContent string
	if data, err := os.ReadFile(securePath); err == nil {
		oldContent = string(data)
	}

	if err := os.WriteFile(securePath, []byte(content), 0644); err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("write file: %v", err)}, nil
	}

	// Generate diff for display
	diff := generateDiff(oldContent, content, securePath)

	return tools.Result{
		Success: true,
		Output:  diff,
		Data: map[string]any{
			"path":    securePath,
			"created": oldContent == "",
			"size":    len(content),
		},
	}, nil
}

// WriteDefinition returns the schema for fs_write.
func WriteDefinition() tools.Definition {
	return tools.Definition{
		Name:        "fs_write",
		Description: "Create or overwrite a file with the given content. Creates parent directories as needed.",
		Parameters: tools.Parameters{
			Type: "object",
			Properties: map[string]tools.Property{
				"path": {
					Type:        "string",
					Description: "Path to the file to write",
				},
				"content": {
					Type:        "string",
					Description: "Content to write to the file",
				},
			},
			Required: []string{"path", "content"},
		},
	}
}

// EditTool implements file editing (find/replace) functionality.
type EditTool struct {
	WorkDir string // Base directory for path validation
}

func (t *EditTool) Name() string        { return "fs_edit" }
func (t *EditTool) Description() string { return "Edit a file by replacing exact text" }

func (t *EditTool) Execute(ctx context.Context, params map[string]any) (tools.Result, error) {
	path, _ := params["path"].(string)
	oldText, _ := params["old_text"].(string)
	newText, _ := params["new_text"].(string)

	if path == "" || oldText == "" {
		return tools.Result{Success: false, Output: fmt.Sprintf("path and old_text parameters are required")}, nil
	}

	// Security check
	baseDir := t.WorkDir
	if baseDir == "" {
		baseDir = GetWorkDir()
	}

	securePath, err := SecurePath(path, baseDir)
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("security check failed: %v", err)}, nil
	}

	data, err := os.ReadFile(securePath)
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("read file: %v", err)}, nil
	}

	content := string(data)
	if !strings.Contains(content, oldText) {
		return tools.Result{Success: false, Output: fmt.Sprintf("old_text not found in file")}, nil
	}

	newContent := strings.Replace(content, oldText, newText, 1)
	if err := os.WriteFile(securePath, []byte(newContent), 0644); err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("write file: %v", err)}, nil
	}

	diff := generateDiff(content, newContent, securePath)

	return tools.Result{
		Success: true,
		Output:  diff,
		Data: map[string]any{
			"path":         securePath,
			"replacements": 1,
		},
	}, nil
}

// EditDefinition returns the schema for fs_edit.
func EditDefinition() tools.Definition {
	return tools.Definition{
		Name:        "fs_edit",
		Description: "Make precise edits to files by replacing exact text. The old_text must match exactly including whitespace.",
		Parameters: tools.Parameters{
			Type: "object",
			Properties: map[string]tools.Property{
				"path": {
					Type:        "string",
					Description: "Path to the file to edit",
				},
				"old_text": {
					Type:        "string",
					Description: "Exact text to find and replace (must match exactly including whitespace)",
				},
				"new_text": {
					Type:        "string",
					Description: "New text to replace the old text with",
				},
			},
			Required: []string{"path", "old_text", "new_text"},
		},
	}
}

// GlobTool implements file pattern matching.
type GlobTool struct{}

func (t *GlobTool) Name() string        { return "fs_glob" }
func (t *GlobTool) Description() string { return "Find files matching a pattern" }

func (t *GlobTool) Execute(ctx context.Context, params map[string]any) (tools.Result, error) {
	pattern, _ := params["pattern"].(string)
	if pattern == "" {
		pattern = "*"
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("glob pattern: %v", err)}, nil
	}

	output := strings.Join(matches, "\n")
	return tools.Result{
		Success: true,
		Output:  output,
		Data: map[string]any{
			"pattern": pattern,
			"matches": matches,
			"count":   len(matches),
		},
	}, nil
}

// GlobDefinition returns the schema for fs_glob.
func GlobDefinition() tools.Definition {
	return tools.Definition{
		Name:        "fs_glob",
		Description: "Find files matching a glob pattern (e.g., '*.go', '**/*.yaml')",
		Parameters: tools.Parameters{
			Type: "object",
			Properties: map[string]tools.Property{
				"pattern": {
					Type:        "string",
					Description: "Glob pattern to match files",
				},
			},
			Required: []string{"pattern"},
		},
	}
}

// GrepTool implements content searching.
type GrepTool struct{}

func (t *GrepTool) Name() string        { return "fs_grep" }
func (t *GrepTool) Description() string { return "Search for patterns in file contents" }

func (t *GrepTool) Execute(ctx context.Context, params map[string]any) (tools.Result, error) {
	pattern, _ := params["pattern"].(string)
	path, _ := params["path"].(string)
	
	if pattern == "" {
		return tools.Result{Success: false, Output: fmt.Sprintf("pattern parameter is required")}, nil
	}
	if path == "" {
		path = "."
	}

	var matches []string
	var count int

	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() > 10*1024*1024 { // Skip files > 10MB
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(line, pattern) {
				matches = append(matches, fmt.Sprintf("%s:%d:%s", file, lineNum, line))
				count++
			}
		}
		return nil
	})

	if err != nil {
		return tools.Result{Success: false, Output: fmt.Sprintf("walk directory: %v", err)}, nil
	}

	output := strings.Join(matches, "\n")
	if output == "" {
		output = "No matches found"
	}

	return tools.Result{
		Success: true,
		Output:  output,
		Data: map[string]any{
			"pattern": pattern,
			"path":    path,
			"count":   count,
		},
	}, nil
}

// GrepDefinition returns the schema for fs_grep.
func GrepDefinition() tools.Definition {
	return tools.Definition{
		Name:        "fs_grep",
		Description: "Search for text patterns in files within a directory",
		Parameters: tools.Parameters{
			Type: "object",
			Properties: map[string]tools.Property{
				"pattern": {
					Type:        "string",
					Description: "Text pattern to search for",
				},
				"path": {
					Type:        "string",
					Description: "Directory path to search in (default: current directory)",
				},
			},
			Required: []string{"pattern"},
		},
	}
}

// generateDiff creates a simple diff view between old and new content.
func generateDiff(oldContent, newContent, path string) string {
	if oldContent == "" {
		// New file
		lines := strings.Split(newContent, "\n")
		var result []string
		result = append(result, path)
		for _, line := range lines {
			result = append(result, "+ "+line)
		}
		return strings.Join(result, "\n")
	}

	// Show simple diff (improved diff can be added later)
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var result []string
	result = append(result, path)
	result = append(result, "")
	result = append(result, "- "+oldLines[0]+"...")
	result = append(result, "+ "+newLines[0]+"...")
	return strings.Join(result, "\n")
}
