package zcontext

import (
	"path/filepath"
	"testing"

	"github.com/mocyuto/zgt/internal/state"
)

func TestNewWithProfileUsesOriginalBranch(t *testing.T) {
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	wtPath := filepath.Join(tmpDir, "repo-hoge")

	state.AppState.Projects = map[string]state.ProjectState{
		"repo": {Worktrees: map[string]*state.WorktreeState{
			wtPath: {Branch: "hoge"},
		}},
	}
	defer func() { state.AppState.Projects = nil }()

	// 作成時ブランチを優先
	ctx := NewWithProfile(wtPath, "fuga", "")
	if ctx.Branch != "hoge" || ctx.TargetBranch != "hoge" {
		t.Errorf("Branch=%q TargetBranch=%q, want hoge", ctx.Branch, ctx.TargetBranch)
	}

	// 記録なしは引数のまま
	ctx = NewWithProfile(filepath.Join(tmpDir, "other"), "fuga", "")
	if ctx.TargetBranch != "fuga" {
		t.Errorf("TargetBranch=%q, want fuga", ctx.TargetBranch)
	}

	// 管理worktree内のネストworktreeは親のブランチを使わない
	ctx = NewWithProfile(filepath.Join(wtPath, ".worktrees", "x"), "x", "")
	if ctx.TargetBranch != "x" {
		t.Errorf("nested TargetBranch=%q, want x", ctx.TargetBranch)
	}
}
