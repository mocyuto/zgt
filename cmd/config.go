package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/template"
	"github.com/mocyuto/zgt/internal/zcontext"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	configCheck  bool
	configGlobal bool
	configLocal  bool
	configRaw    bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display the merged configuration",
	Long:  "Display the merged configuration from global, project, and command-line sources in YAML format.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configCheck {
			if config.ConfigError != nil {
				return config.ConfigError
			}
			logger.Success("Configuration is valid.")
			return nil
		}

		// Prepare context for placeholder replacement
		cwd, _ := os.Getwd()
		_ = state.LoadState()
		ctx := zcontext.New(cwd, "")

		var replacedConfig config.Config
		header := "--- Merged Configuration (Placeholders Replaced) ---"
		if configRaw {
			replacedConfig = config.AppConfig
			header = "--- Merged Configuration (Raw) ---"
		} else {
			replacedConfig = template.ReplaceConfig(config.AppConfig, ctx)
		}

		data, err := yaml.Marshal(replacedConfig)
		if err != nil {
			return logger.Errorf("error marshaling config: %v", err)
		}

		cmd.Println(header)
		cmd.Print(string(data))
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the configuration file",
	Long:  "Edit the global or local configuration file using the system editor.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var path string
		if configGlobal {
			var err error
			path, err = config.GetGlobalConfigPath()
			if err != nil {
				return logger.Errorf("could not get global config path: %v", err)
			}
		} else {
			// Default or --local: target local config
			gitRoot, err := gitroot.GetMainProjectRoot()
			if err != nil || gitRoot == "" {
				return logger.Errorf("not in a git repository or could not find project root")
			}
			path = config.GetLocalConfigPath(gitRoot)
			if path == "" {
				// If not exists, use default name for creation/edit
				path = filepath.Join(gitRoot, config.DefaultLocalConfigName)
			}
		}

		if path == "" {
			return logger.Errorf("could not determine configuration file path")
		}

		// Ensure directory exists for global config
		if configGlobal {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return logger.Errorf("failed to create config directory: %v", err)
			}
		}

		return editFile(path)
	},
}

func editFile(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Read original content
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return logger.Errorf("failed to read config file: %v", err)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "zgt-config-*.yaml")
	if err != nil {
		return logger.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		return logger.Errorf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	for {
		cmd := exec.Command(editor, tmpFile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return logger.Errorf("editor failed: %v", err)
		}

		// Validate YAML
		newContent, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return logger.Errorf("failed to read edited file: %v", err)
		}

		var cfg config.Config
		if err := yaml.Unmarshal(newContent, &cfg); err != nil {
			fmt.Printf("YAML validation failed: %v\n", err)
			fmt.Print("Press Enter to re-edit or Ctrl+C to abort...")
			fmt.Scanln()
			continue
		}

		// Write back to original file
		if err := os.WriteFile(path, newContent, 0644); err != nil {
			return logger.Errorf("failed to save config file: %v", err)
		}

		logger.Success("Configuration saved to %s", path)
		break
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVar(&configCheck, "check", false, "check configuration for errors")
	configCmd.Flags().BoolVar(&configRaw, "raw", false, "display configuration without placeholder replacement")

	configCmd.AddCommand(configEditCmd)
	configEditCmd.Flags().BoolVarP(&configGlobal, "global", "g", false, "edit global configuration")
	configEditCmd.Flags().BoolVarP(&configLocal, "local", "l", false, "edit local configuration")
}
