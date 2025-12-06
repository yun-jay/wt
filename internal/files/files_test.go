package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSymlink(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "wt-files-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sourceWT := filepath.Join(tmpDir, "source-wt")
	targetWT := filepath.Join(tmpDir, "target-wt")

	if err := os.MkdirAll(sourceWT, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetWT, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Create source file
	sourceFile := filepath.Join(sourceWT, ".env")
	if err := os.WriteFile(sourceFile, []byte("SECRET=123"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create symlink
	err = createSymlink(sourceWT, targetWT, ".env")
	if err != nil {
		t.Fatalf("createSymlink failed: %v", err)
	}

	// Verify symlink exists
	targetFile := filepath.Join(targetWT, ".env")
	info, err := os.Lstat(targetFile)
	if err != nil {
		t.Fatalf("failed to stat target file: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("target is not a symlink")
	}

	// Verify symlink points to source
	link, err := os.Readlink(targetFile)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	if link != sourceFile {
		t.Errorf("symlink target = %s, want %s", link, sourceFile)
	}
}

func TestCreateSymlinkNonExistentSource(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-files-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sourceWT := filepath.Join(tmpDir, "source-wt")
	targetWT := filepath.Join(tmpDir, "target-wt")

	if err := os.MkdirAll(sourceWT, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetWT, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Try to symlink non-existent file - should succeed silently
	err = createSymlink(sourceWT, targetWT, "non-existent.txt")
	if err != nil {
		t.Errorf("createSymlink should not error for non-existent source: %v", err)
	}

	// Verify no symlink was created
	targetFile := filepath.Join(targetWT, "non-existent.txt")
	if _, err := os.Lstat(targetFile); !os.IsNotExist(err) {
		t.Error("symlink should not be created for non-existent source")
	}
}

func TestCreateSymlinkNestedDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-files-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sourceWT := filepath.Join(tmpDir, "source-wt")
	targetWT := filepath.Join(tmpDir, "target-wt")

	// Create nested source directory
	nestedDir := filepath.Join(sourceWT, "config", "settings")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	if err := os.MkdirAll(targetWT, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Create source file in nested dir
	sourceFile := filepath.Join(nestedDir, "app.yaml")
	if err := os.WriteFile(sourceFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create symlink
	err = createSymlink(sourceWT, targetWT, "config/settings/app.yaml")
	if err != nil {
		t.Fatalf("createSymlink failed: %v", err)
	}

	// Verify symlink exists
	targetFile := filepath.Join(targetWT, "config", "settings", "app.yaml")
	info, err := os.Lstat(targetFile)
	if err != nil {
		t.Fatalf("failed to stat target file: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("target is not a symlink")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-files-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sourceWT := filepath.Join(tmpDir, "source-wt")
	targetWT := filepath.Join(tmpDir, "target-wt")

	if err := os.MkdirAll(sourceWT, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetWT, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Create source file
	sourceFile := filepath.Join(sourceWT, "config.json")
	content := []byte(`{"key": "value"}`)
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Copy file
	err = copyFile(sourceWT, targetWT, "config.json")
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify copy exists and has correct content
	targetFile := filepath.Join(targetWT, "config.json")
	copiedContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}

	if string(copiedContent) != string(content) {
		t.Errorf("copied content = %s, want %s", copiedContent, content)
	}

	// Verify it's a regular file (not symlink)
	info, err := os.Lstat(targetFile)
	if err != nil {
		t.Fatalf("failed to stat target file: %v", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("target should be a regular file, not a symlink")
	}
}

func TestCopyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-files-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sourceWT := filepath.Join(tmpDir, "source-wt")
	targetWT := filepath.Join(tmpDir, "target-wt")

	// Create source directory with files
	sourceDir := filepath.Join(sourceWT, "templates")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetWT, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Create files in source directory
	files := map[string]string{
		"index.html": "<html></html>",
		"style.css":  "body {}",
	}
	for name, content := range files {
		path := filepath.Join(sourceDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
	}

	// Copy directory
	err = copyFile(sourceWT, targetWT, "templates")
	if err != nil {
		t.Fatalf("copyFile (dir) failed: %v", err)
	}

	// Verify all files were copied
	for name, expectedContent := range files {
		targetPath := filepath.Join(targetWT, "templates", name)
		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("failed to read copied file %s: %v", name, err)
		}
		if string(content) != expectedContent {
			t.Errorf("file %s content = %s, want %s", name, content, expectedContent)
		}
	}
}

func TestSetupFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-files-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sourceWT := filepath.Join(tmpDir, "source-wt")
	targetWT := filepath.Join(tmpDir, "target-wt")

	if err := os.MkdirAll(sourceWT, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetWT, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Create source files
	os.WriteFile(filepath.Join(sourceWT, ".env"), []byte("SECRET=123"), 0644)
	os.WriteFile(filepath.Join(sourceWT, "config.json"), []byte("{}"), 0644)

	// Setup files
	symlinks := []string{".env"}
	copies := []string{"config.json"}

	err = SetupFiles(sourceWT, targetWT, symlinks, copies)
	if err != nil {
		t.Fatalf("SetupFiles failed: %v", err)
	}

	// Verify .env is a symlink
	envInfo, err := os.Lstat(filepath.Join(targetWT, ".env"))
	if err != nil {
		t.Fatalf("failed to stat .env: %v", err)
	}
	if envInfo.Mode()&os.ModeSymlink == 0 {
		t.Error(".env should be a symlink")
	}

	// Verify config.json is a regular file
	configInfo, err := os.Lstat(filepath.Join(targetWT, "config.json"))
	if err != nil {
		t.Fatalf("failed to stat config.json: %v", err)
	}
	if configInfo.Mode()&os.ModeSymlink != 0 {
		t.Error("config.json should be a regular file, not a symlink")
	}
}

func TestCleanupSymlinks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-files-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	deletedWT := filepath.Join(tmpDir, "deleted-wt")
	otherWT := filepath.Join(tmpDir, "other-wt")

	if err := os.MkdirAll(deletedWT, 0755); err != nil {
		t.Fatalf("failed to create deleted dir: %v", err)
	}
	if err := os.MkdirAll(otherWT, 0755); err != nil {
		t.Fatalf("failed to create other dir: %v", err)
	}

	// Create a file in deleted worktree
	deletedFile := filepath.Join(deletedWT, ".env")
	if err := os.WriteFile(deletedFile, []byte("SECRET=123"), 0644); err != nil {
		t.Fatalf("failed to create deleted file: %v", err)
	}

	// Create a symlink in other worktree pointing to deleted worktree
	symlinkPath := filepath.Join(otherWT, ".env")
	if err := os.Symlink(deletedFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Verify symlink exists before cleanup
	if _, err := os.Lstat(symlinkPath); err != nil {
		t.Fatalf("symlink should exist before cleanup: %v", err)
	}

	// Cleanup symlinks
	CleanupSymlinks(deletedWT, []string{otherWT})

	// Verify symlink was removed
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Error("symlink should be removed after cleanup")
	}
}
