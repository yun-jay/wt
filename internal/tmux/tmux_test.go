package tmux

import (
	"os"
	"testing"

	"github.com/yunus/wt/internal/config"
)

func TestIsInsideTmux(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer os.Setenv("TMUX", originalTmux)

	// Test when not in tmux
	os.Unsetenv("TMUX")
	if IsInsideTmux() {
		t.Error("IsInsideTmux() = true, want false when TMUX is not set")
	}

	// Test when in tmux
	os.Setenv("TMUX", "/tmp/tmux-501/default,12345,0")
	if !IsInsideTmux() {
		t.Error("IsInsideTmux() = false, want true when TMUX is set")
	}
}

func TestGetCurrentSessionNotInTmux(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer os.Setenv("TMUX", originalTmux)

	os.Unsetenv("TMUX")

	_, err := GetCurrentSession()
	if err != ErrNotInTmux {
		t.Errorf("GetCurrentSession() error = %v, want ErrNotInTmux", err)
	}
}

func TestGetCurrentWindowNotInTmux(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer os.Setenv("TMUX", originalTmux)

	os.Unsetenv("TMUX")

	_, err := GetCurrentWindow()
	if err != ErrNotInTmux {
		t.Errorf("GetCurrentWindow() error = %v, want ErrNotInTmux", err)
	}
}

func TestGetCurrentWindowNameNotInTmux(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer os.Setenv("TMUX", originalTmux)

	os.Unsetenv("TMUX")

	_, err := GetCurrentWindowName()
	if err != ErrNotInTmux {
		t.Errorf("GetCurrentWindowName() error = %v, want ErrNotInTmux", err)
	}
}

func TestSessionExistsNonExistent(t *testing.T) {
	// Test with a session name that definitely doesn't exist
	exists := SessionExists("wt-test-nonexistent-session-12345")
	if exists {
		t.Error("SessionExists() = true for non-existent session")
	}
}

func TestToggleNotInTmux(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer os.Setenv("TMUX", originalTmux)

	os.Unsetenv("TMUX")

	cfg := &config.Config{
		TogglePane: &config.TogglePaneConfig{
			Name:     "test",
			Command:  "echo test",
			Position: "right",
			Size:     20,
		},
	}

	err := Toggle(cfg)
	if err != ErrNotInTmux {
		t.Errorf("Toggle() error = %v, want ErrNotInTmux", err)
	}
}

func TestToggleNoConfig(t *testing.T) {
	// Save original TMUX env var and set it
	originalTmux := os.Getenv("TMUX")
	defer os.Setenv("TMUX", originalTmux)

	os.Setenv("TMUX", "/tmp/tmux-501/default,12345,0")

	cfg := &config.Config{
		TogglePane: nil,
	}

	err := Toggle(cfg)
	if err == nil {
		t.Error("Toggle() should error when no toggle pane configured")
	}
}

func TestIsTogglePaneVisibleNotInTmux(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer os.Setenv("TMUX", originalTmux)

	os.Unsetenv("TMUX")

	cfg := &config.Config{
		TogglePane: &config.TogglePaneConfig{
			Name: "test",
		},
	}

	_, err := IsTogglePaneVisible(cfg)
	if err != ErrNotInTmux {
		t.Errorf("IsTogglePaneVisible() error = %v, want ErrNotInTmux", err)
	}
}

func TestSwitchSessionNotFound(t *testing.T) {
	err := SwitchSession("wt-test-nonexistent-session-12345")
	if err != ErrSessionNotFound {
		t.Errorf("SwitchSession() error = %v, want ErrSessionNotFound", err)
	}
}

// Integration tests - only run inside tmux
func TestIntegrationCreateAndKillSession(t *testing.T) {
	if !IsInsideTmux() {
		t.Skip("Skipping integration test: not running inside tmux")
	}

	sessionName := "wt-test-integration-session"
	cfg := config.DefaultConfig()

	// Create temp dir for session
	tmpDir := t.TempDir()

	// Create session
	err := CreateSession(sessionName, tmpDir, cfg)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify session exists
	if !SessionExists(sessionName) {
		t.Error("Session should exist after creation")
	}

	// Kill session
	err = KillSession(sessionName)
	if err != nil {
		t.Fatalf("KillSession failed: %v", err)
	}

	// Verify session is gone
	if SessionExists(sessionName) {
		t.Error("Session should not exist after kill")
	}
}

func TestIntegrationListSessions(t *testing.T) {
	if !IsInsideTmux() {
		t.Skip("Skipping integration test: not running inside tmux")
	}

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	// Should have at least one session (the current one)
	if len(sessions) == 0 {
		t.Error("ListSessions returned empty list, expected at least one session")
	}
}
