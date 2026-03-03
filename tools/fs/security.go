package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SecurePath validates and sanitizes a file path to prevent directory traversal attacks.
// It ensures the resolved path is within the allowed base directory.
func SecurePath(path, baseDir string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	// Clean the path to resolve . and ..
	path = filepath.Clean(path)

	// Get absolute paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base directory: %w", err)
	}

	// Ensure base directory ends with separator for proper prefix checking
	if !strings.HasSuffix(absBase, string(filepath.Separator)) {
		absBase += string(filepath.Separator)
	}

	// Check if the path is within the allowed base directory
	// We add a separator to absPath to prevent partial directory name matching
	checkPath := absPath
	if !strings.HasSuffix(checkPath, string(filepath.Separator)) {
		checkPath += string(filepath.Separator)
	}

	if !strings.HasPrefix(checkPath, absBase) && absPath != strings.TrimSuffix(absBase, string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes base directory: %s is not within %s", absPath, absBase)
	}

	// Check for symlink traversal
	fi, err := os.Lstat(absPath)
	if err == nil {
		// Path exists, check if it's a symlink
		if fi.Mode()&os.ModeSymlink != 0 {
			// Resolve symlink and verify target
			target, err := os.Readlink(absPath)
			if err != nil {
				return "", fmt.Errorf("failed to read symlink: %w", err)
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(absPath), target)
			}
			target = filepath.Clean(target)
			
			// Verify symlink target is within base directory
			targetCheck := target
			if !strings.HasSuffix(targetCheck, string(filepath.Separator)) {
				targetCheck += string(filepath.Separator)
			}
			if !strings.HasPrefix(targetCheck, absBase) {
				return "", fmt.Errorf("symlink target escapes base directory")
			}
		}
	}

	return absPath, nil
}

// GetWorkDir returns the current working directory or "." if unavailable.
func GetWorkDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// IsPathAllowed checks if a path is allowed to be accessed.
// It performs the same checks as SecurePath but returns a boolean.
func IsPathAllowed(path, baseDir string) bool {
	_, err := SecurePath(path, baseDir)
	return err == nil
}
