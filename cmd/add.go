package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/hook"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/mocyuto/zgt/internal/zcontext"
	"github.com/spf13/cobra"
)

var (
	baseFlag        string
	fromDefaultFlag bool
	pathFlag        string
	profileFlag     string
)

var addCmd = &cobra.Command{
	Use:   "add <branch>",
	Short: "Create git worktree and copy ignored files",
	Long: `Create a new git worktree, optionally creating a new branch, and
automatically copy ignored configuration files (like .env) from the main tree.

The argument is treated as the branch name. By default, the worktree path
is automatically determined based on the main repository root.
Use the --path flag to specify a custom target path.
Use the --profile flag to select a profile defined under the "profiles"
section of zgt.config.yml. Profiles override env vars and append hooks
for specialized workflows (e.g., spinning up an isolated DB).

Both forms will automatically create the branch if it does not already exist.`,
	Example: `  # Automated path: if repo root is 'path-to/myapp', creates worktree at 'path-to/myapp-feat'
  zgt add feat

  # Explicit path:
  zgt add feat --path ./experimental-worktree

  # Use the "migration" profile to spin up an isolated DB worktree:
  zgt add feat/migrate-db --profile migration`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]
		var targetPath string
		mainRoot, err := gitroot.GetMainProjectRoot()
		if err != nil {
			return logger.Errorf("failed to get main project root: %v", err)
		}

		// Validate profile if specified
		if profileFlag != "" && !config.ProfileExists(profileFlag) {
			return logger.Errorf("unknown profile %q; define it under the 'profiles' section in zgt.config.yml", profileFlag)
		}

		if pathFlag != "" {
			targetPath = pathFlag
		} else {
			projectName := filepath.Base(mainRoot)
			targetPath = filepath.Join(filepath.Dir(mainRoot), fmt.Sprintf("%s-%s", projectName, branch))
			logger.Info("Automated path: %s", targetPath)
		}

		sourceRoot, err := gitroot.GetGitRoot()
		if err != nil {
			return logger.Errorf("failed to get git root: %v", err)
		}

		var baseBranch string
		useDefault := fromDefaultFlag || config.AppConfig.Add.FromDefault
		if baseFlag != "" {
			baseBranch = baseFlag
			logger.Info("Using specified branch '%s' as base", baseBranch)
		} else if useDefault {
			defaultBranch, err := git.GetDefaultBranch()
			if err != nil {
				return logger.Errorf("failed to get default branch: %v", err)
			}
			baseBranch = defaultBranch
			logger.Info("Using default branch '%s' as base", baseBranch)

			if config.AppConfig.Add.AutoPull {
				// Determine which directory to update
				updateDir := "."
				wtPath, err := git.GetBranchWorktree(baseBranch)
				if err == nil && wtPath != "" {
					updateDir = wtPath
				}

				if err := git.PullWithAutostash(updateDir, baseBranch); err != nil {
					logger.Warn("Failed to update branch '%s': %v", baseBranch, err)
					logger.Info("Falling back to fetch from remote...")
					if err := git.Fetch(baseBranch); err != nil {
						logger.Warn("Fetch failed: %v", err)
					} else {
						baseBranch = "origin/" + baseBranch
						logger.Success("Using '%s' as base for the new worktree", baseBranch)
					}
				} else {
					logger.Success("Successfully updated '%s'", baseBranch)
				}
			}
		} else {
			logger.Info("Using current branch as base")
		}

		logger.Info("--- Creating worktree at %s ---", targetPath)
		if err := git.CreateWorktree(targetPath, branch, baseBranch); err != nil {
			return logger.Errorf("error creating worktree: %v", err)
		}

		logger.Info("--- Copying ignored configuration files ---")
		if err := git.CopyIgnoredFiles(sourceRoot, targetPath, config.AppConfig.Ignore, logger.Verbose); err != nil {
			return logger.Errorf("error copying files: %v", err)
		}

		// Assign port index and persist profile EARLY so subsequent
		// steps (git hooks, tmux, hooks) - even if they fail - leave the
		// worktree in a recoverable state in zgt's state file.
		absPath := state.NormalizePath(targetPath)
		_ = state.LoadState() // Assign ports
		projectName := filepath.Base(mainRoot)
		for name, basePort := range config.AppConfig.Ports {
			state.AssignPortIndex(projectName, absPath, name, basePort)
		}
		if profileFlag != "" {
			state.SetProfile(projectName, absPath, profileFlag)
		}
		// worktree内でブランチ切替後もテンプレート・rmで使う作成時ブランチ
		state.SetBranch(projectName, absPath, branch)
		_ = state.SaveState()

		if config.AppConfig.GitHooks.Enabled {
			hooksPath := ""
			var err error

			if config.AppConfig.GitHooks.Shared {
				// Shared: Resolve from main root and use absolute path
				hooksPath, err = git.ResolveHooksPath(mainRoot, config.AppConfig.GitHooks.Path)
				if err != nil {
					return logger.Errorf("error resolving git hooks path: %v", err)
				}
			} else {
				// Not shared: Copy from source to target
				srcHooks, err := git.ResolveHooksPath(sourceRoot, config.AppConfig.GitHooks.Path)
				if err != nil {
					return logger.Errorf("error resolving source git hooks path: %v", err)
				}

				absTarget, err := filepath.Abs(targetPath)
				if err != nil {
					return logger.Errorf("failed to get absolute path for target: %v", err)
				}
				dstHooks, err := git.ResolveHooksPath(absTarget, config.AppConfig.GitHooks.Path)
				if err != nil {
					return logger.Errorf("error resolving target git hooks path: %v", err)
				}

				// Copy if source exists
				if info, err := os.Stat(srcHooks); err == nil && info.IsDir() {
					logger.Info("--- Copying git hooks from %s to %s ---", srcHooks, dstHooks)
					if err := git.CopyHooksDir(srcHooks, dstHooks); err != nil {
						return logger.Errorf("error copying git hooks: %v", err)
					}
				}
				hooksPath = dstHooks
			}

			logger.Info("--- Registering git hooks at %s ---", hooksPath)
			registered, existingPath, err := git.RegisterHooksPath(targetPath, hooksPath)
			if err != nil {
				return logger.Errorf("error registering git hooks: %v", err)
			}
			if registered {
				logger.Success("Registered git hooks for worktree")
			} else {
				logger.Warn("git hooks already configured for worktree (%s); skipping", existingPath)
			}
		}

		logger.Success("--- Done! ---")
		logger.Success("New worktree is ready at: %s", targetPath)

		// Run tmux setup
		ctx := zcontext.NewWithProfile(absPath, branch, profileFlag)
		if err := tmux.Setup(ctx); err != nil {
			logger.Warn("tmux setup failed: %v", err)
		}

		// Run add hooks (profile-aware)
		hook.RunHooks("add", ctx)

		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&baseFlag, "base", "b", "", "specify a base branch to create the worktree from")
	addCmd.Flags().BoolVar(&fromDefaultFlag, "from-default", false, "use the default branch as base")
	addCmd.Flags().StringVarP(&pathFlag, "path", "p", "", "custom target path for the worktree")
	addCmd.Flags().StringVarP(&profileFlag, "profile", "", "", "select a profile from the 'profiles' section of zgt.config.yml")
	rootCmd.AddCommand(addCmd)

	// Shell completion for --profile: list profile names defined in config.
	// config.InitConfig runs via cobra.OnInitialize before completion funcs,
	// so AppConfig.Profiles is populated by the time this is invoked.
	_ = addCmd.RegisterFlagCompletionFunc("profile", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return config.ProfileNames(), cobra.ShellCompDirectiveNoFileComp
	})
}
