package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize wt configuration",
	Long: `Initialize wt configuration.

When run without arguments or outside a git repo, creates the global config.
When run in a git bare repo, creates a project-specific .wt.yaml file.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repo (has projectRoot set)
	if projectRoot != "" {
		return initProjectConfig()
	}
	return initGlobalConfig()
}

func initGlobalConfig() error {
	configPath := config.GlobalConfigPath()

	// Check if already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Global config already exists at %s\n", configPath)
		return nil
	}

	// Create directory
	if err := config.EnsureGlobalConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config
	cfg := config.DefaultConfig()
	if err := cfg.SaveGlobal(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Created global config at %s\n", configPath)
	printExampleConfig()
	return nil
}

func initProjectConfig() error {
	configPath := filepath.Join(projectRoot, config.ProjectConfigFile)

	// Check if already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Project config already exists at %s\n", configPath)
		return nil
	}

	// Create example project config
	projectCfg := &config.Config{
		Windows: []config.Window{
			{
				Name:  "nvim",
				Focus: true,
				Panes: []config.Pane{
					{Name: "editor", Command: "nvim ."},
				},
			},
			{
				Name: "services",
				Panes: []config.Pane{
					{Name: "service1", Command: "# npm run dev"},
					{Name: "service2", Command: "# npm run api", Split: "horizontal", Size: 50},
				},
			},
			{
				Name: "test",
				Panes: []config.Pane{
					{Name: "test", Command: "# npm test --watch"},
				},
			},
		},
		TogglePane: &config.TogglePaneConfig{
			Name:     "claude",
			Command:  "<agent>",
			Position: "right",
			Size:     20,
		},
		PostCreate: []string{"<global>"},
		Files: config.Files{
			Symlink: []string{"<global>"},
		},
	}

	if err := projectCfg.SaveProject(projectRoot); err != nil {
		return fmt.Errorf("failed to save project config: %w", err)
	}

	fmt.Printf("Created project config at %s\n", configPath)
	printExampleProjectConfig()
	return nil
}

func printExampleConfig() {
	fmt.Print(`
Example global config (~/.config/wt/config.yaml):

windows:
  - name: nvim
    focus: true
    panes:
      - name: editor
        command: nvim .

  - name: shell
    panes:
      - name: shell

toggle_pane:
  name: claude
  command: claude
  position: right
  size: 20

agent: claude
`)
}

func printExampleProjectConfig() {
	fmt.Print(`
Example project config (.wt.yaml):

windows:
  - name: nvim
    focus: true
    panes:
      - name: editor
        command: nvim .

  - name: services
    panes:
      - name: api
        command: npm run api
      - name: web
        command: npm run dev
        split: horizontal
        size: 50

  - name: test
    panes:
      - name: test
        command: npm test --watch

toggle_pane:
  name: claude
  command: <agent>
  position: right
  size: 20
  hooks:
    on_open: "echo 'Claude opened'"
    on_close: "echo 'Claude closed'"

post_create:
  - '<global>'
  - npm install
`)
}
