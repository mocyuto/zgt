package cmd

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"

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
	forceDelete bool
	keepBranch  bool
)

var removeCmd = &cobra.Command{
	Use:     "remove <branch>",
	Aliases: []string{"rm"},
	Short:   "Remove a worktree by branch name",
	Args:    cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		wts, err := git.GetWorktrees()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		mainRoot, _ := gitroot.GetMainProjectRoot()
		_ = state.LoadState()
		return removeCompletions(wts, state.AppState.Projects[filepath.Base(mainRoot)]), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		branchOrPath := args[0]
		var path, branch string
		mainRoot, _ := gitroot.GetMainProjectRoot()
		_ = state.LoadState()

		// worktree内でブランチ切り替え済みのケース（サフィックス一致より優先）
		_, exactErr := git.GetBranchWorktree(branchOrPath)
		mp, mb, mismatched := findMismatchedWorktree(mainRoot, branchOrPath)
		if exactErr != nil && mismatched {
			branch = branchOrPath
			msg := fmt.Sprintf("Worktree %s now has branch '%s' checked out (not '%s'). Remove the folder too? [y/N]: ", mp, mb, branchOrPath)
			yes, err := confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), msg)
			if err != nil {
				return logger.Errorf("aborted: no answer to prompt (pipe 'y' or 'n')")
			}
			if !yes {
				// worktreeは現ブランチの扱いへ（stale参照防止）
				state.ClearBranch(filepath.Base(mainRoot), mp)
				_ = state.SaveState()
				if !keepBranch {
					logger.Info("--- Deleting branch %s ---", branch)
					if err := git.DeleteBranch(branch); err != nil {
						logger.Warn("failed to delete branch %s: %v", branch, err)
					}
				}
				logger.Success("--- Done! (worktree kept) ---")
				return nil
			}
			path = mp
		} else if p, b, err := git.ResolveWorktreeInfo(branchOrPath); err == nil {
			path = p
			branch = b
		} else {
			logger.Info("Notice: Could not resolve worktree for '%s' in list, trying as path...", branchOrPath)
			path = branchOrPath
			// If it was a path, we might still want to find its branch for deletion
			_, branch, _ = git.ResolveWorktreeInfo(path)
		}

		// Resolve profile from state BEFORE removing the worktree entry,
		// so that profile-specific rm hooks (e.g. isolated DB cleanup) run.
		absPath := state.NormalizePath(path)
		projectName := filepath.Base(mainRoot)
		profile := ""
		if proj, ok := state.AppState.Projects[projectName]; ok {
			if wt, ok := proj.Worktrees[absPath]; ok {
				profile = wt.Profile
			}
		}

		// Gracefully close tmux window if configured
		ctx := zcontext.NewWithProfile(path, branch, profile)
		if err := tmux.CloseWindow(ctx, false); err != nil {
			logger.Warn("Failed to close tmux window: %v", err)
		}

		logger.Info("--- Removing worktree at %s ---", path)
		if err := git.RemoveWorktree(path, forceDelete); err != nil {
			return logger.Errorf("error removing worktree: %v", err)
		}

		// Delete branch if not kept
		if !keepBranch && branch != "" {
			logger.Info("--- Deleting branch %s ---", branch)
			if err := git.DeleteBranch(branch); err != nil {
				logger.Warn("failed to delete branch %s: %v", branch, err)
			}
		}

		// Run removal hooks (profile-aware)
		hook.RunHooks("rm", ctx)

		// Release port index
		state.ReleasePortIndex(projectName, absPath)
		state.CleanupState()
		_ = state.SaveState()

		logger.Success("--- Done!---")
		return nil
	},
}

// 作成時ブランチ（state、なければ自動パス）から、別ブランチがチェックアウトされたworktreeの検出
func findMismatchedWorktree(mainRoot, branch string) (path, actualBranch string, ok bool) {
	if mainRoot == "" {
		return "", "", false
	}
	candidates := append(state.FindPathsByBranch(filepath.Base(mainRoot), branch),
		filepath.Join(filepath.Dir(mainRoot), fmt.Sprintf("%s-%s", filepath.Base(mainRoot), branch)))
	for _, c := range candidates {
		p, b, err := git.ResolveWorktreeInfo(c)
		if err == nil && b != branch {
			return p, b, true
		}
	}
	return "", "", false
}

// 現在のブランチ＋ブランチ切替済みworktreeの作成時ブランチ（説明付き）
func removeCompletions(wts []git.Worktree, proj state.ProjectState) []string {
	var res []string
	current := make(map[string]bool)
	for _, wt := range wts {
		if wt.Branch == "" {
			continue
		}
		b := strings.TrimPrefix(wt.Branch, "refs/heads/")
		current[b] = true
		res = append(res, b)
	}
	for _, wt := range wts {
		if wt.Branch == "" {
			continue // detached HEADはrmで解決不可
		}
		ws, ok := proj.Worktrees[state.NormalizePath(wt.Path)]
		if !ok || ws.Branch == "" || current[ws.Branch] {
			continue
		}
		now := strings.TrimPrefix(wt.Branch, "refs/heads/")
		res = append(res, fmt.Sprintf("%s\tworktree %s (now on %s)", ws.Branch, wt.Path, now))
		current[ws.Branch] = true
	}
	return res
}

// y/yes のみ true、入力なしEOFはエラー
func confirm(in io.Reader, out io.Writer, msg string) (bool, error) {
	fmt.Fprint(out, msg)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

func init() {
	removeCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "force removal even if worktree is dirty")
	removeCmd.Flags().BoolVarP(&keepBranch, "keep-branch", "k", false, "do not delete the associated branch")
	rootCmd.AddCommand(removeCmd)
}
