package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SetupFiles creates symlinks and copies files for a new worktree
func SetupFiles(sourceWorktree, targetWorktree string, symlinks, copies []string) error {
	// Handle symlinks
	for _, file := range symlinks {
		if err := createSymlink(sourceWorktree, targetWorktree, file); err != nil {
			// Log but don't fail on symlink errors
			fmt.Printf("Warning: failed to symlink %s: %v\n", file, err)
		}
	}

	// Handle copies
	for _, file := range copies {
		if err := copyFile(sourceWorktree, targetWorktree, file); err != nil {
			// Log but don't fail on copy errors
			fmt.Printf("Warning: failed to copy %s: %v\n", file, err)
		}
	}

	return nil
}

// createSymlink creates a symlink from source to target worktree
func createSymlink(sourceWorktree, targetWorktree, file string) error {
	sourcePath := filepath.Join(sourceWorktree, file)
	targetPath := filepath.Join(targetWorktree, file)

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil // Source doesn't exist, skip silently
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// Remove target if it exists
	os.Remove(targetPath)

	// Create symlink
	return os.Symlink(sourcePath, targetPath)
}

// copyFile copies a file or directory from source to target worktree
func copyFile(sourceWorktree, targetWorktree, file string) error {
	sourcePath := filepath.Join(sourceWorktree, file)
	targetPath := filepath.Join(targetWorktree, file)

	// Check if source exists
	info, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		return nil // Source doesn't exist, skip silently
	}
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(sourcePath, targetPath)
	}

	return copyFileContents(sourcePath, targetPath)
}

// copyFileContents copies a single file's contents
func copyFileContents(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFileContents(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// CleanupSymlinks removes symlinks that point to a deleted worktree
func CleanupSymlinks(deletedWorktree string, otherWorktrees []string) {
	for _, wt := range otherWorktrees {
		entries, err := os.ReadDir(wt)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			path := filepath.Join(wt, entry.Name())
			if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err == nil && filepath.HasPrefix(target, deletedWorktree) {
					os.Remove(path)
				}
			}
		}
	}
}
