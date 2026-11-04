package zcontext

import (
	"path/filepath"

	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/template"
)

// New creates a new template.Context with populated fields.
// path is the target path (e.g. worktree path).
// targetBranch is the branch being operated on.
func New(path, targetBranch string) template.Context {
	return NewWithProfile(path, targetBranch, "")
}

// NewWithProfile creates a new template.Context with the given profile name.
func NewWithProfile(path, targetBranch, profile string) template.Context {
	mainRoot, _ := gitroot.GetMainProjectRoot()
	currentRoot, _ := gitroot.GetGitRoot()
	currentBranch, _ := git.GetCurrentBranch()

	// zgt add 時のブランチを優先（worktree内でのブランチ切替対策）
	// branch未指定（env/config）はcwdのgit rootで照合
	lookup := path
	if targetBranch == "" {
		lookup = currentRoot
	}
	if original := state.OriginalBranch(lookup); original != "" {
		targetBranch = original
	}

	// If targetBranch is empty (e.g. in config or env commands), use currentBranch
	if targetBranch == "" {
		targetBranch = currentBranch
	}

	return template.Context{
		Path:          path,
		Branch:        targetBranch,
		Repo:          filepath.Base(mainRoot),
		CurrentDir:    filepath.Base(currentRoot),
		TargetBranch:  targetBranch,
		CurrentBranch: currentBranch,
		Profile:       profile,
	}
}
