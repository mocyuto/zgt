package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	cases := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{"y\n", true, false},
		{"Y\n", true, false},
		{"yes\n", true, false},
		{" YES \n", true, false},
		{"y", true, false},
		{"n\n", false, false},
		{"\n", false, false},
		{"anything\n", false, false},
		{"", false, true},
	}
	for _, c := range cases {
		var out bytes.Buffer
		got, err := confirm(strings.NewReader(c.input), &out, "Remove? [y/N]: ")
		if got != c.want || (err != nil) != c.wantErr {
			t.Errorf("confirm(%q) = %v, %v; want %v, wantErr %v", c.input, got, err, c.want, c.wantErr)
		}
		if out.String() != "Remove? [y/N]: " {
			t.Errorf("unexpected prompt output: %q", out.String())
		}
	}
}

func TestFindMismatchedWorktree(t *testing.T) {
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Join(tmpDir, "myrepo")
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "init", "-b", "main")
	runGit(t, repoRoot, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "--allow-empty", "-m", "init")

	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}

	wtPath := filepath.Join(tmpDir, "myrepo-feat-a")
	runGit(t, repoRoot, "worktree", "add", "-b", "feat-a", wtPath)

	t.Run("BranchMatchesFolder", func(t *testing.T) {
		if _, _, ok := findMismatchedWorktree(repoRoot, "feat-a"); ok {
			t.Error("expected no mismatch when branch matches folder")
		}
	})

	t.Run("NoWorktreeForBranch", func(t *testing.T) {
		if _, _, ok := findMismatchedWorktree(repoRoot, "unknown"); ok {
			t.Error("expected no mismatch for unknown branch")
		}
	})

	t.Run("EmptyMainRoot", func(t *testing.T) {
		if _, _, ok := findMismatchedWorktree("", "feat-a"); ok {
			t.Error("expected no mismatch for empty main root")
		}
	})

	// worktree内で別ブランチに切り替え
	runGit(t, wtPath, "checkout", "-b", "feat-b")

	t.Run("BranchSwitchedInWorktree", func(t *testing.T) {
		path, branch, ok := findMismatchedWorktree(repoRoot, "feat-a")
		if !ok {
			t.Fatal("expected mismatch to be detected")
		}
		if path != wtPath {
			t.Errorf("expected path %s, got %s", wtPath, path)
		}
		if branch != "feat-b" {
			t.Errorf("expected branch feat-b, got %s", branch)
		}
	})

	// サフィックス一致するブランチ名への切り替え
	runGit(t, wtPath, "checkout", "-b", "feature/feat-a")

	t.Run("SwitchedToSuffixMatchingBranch", func(t *testing.T) {
		_, branch, ok := findMismatchedWorktree(repoRoot, "feat-a")
		if !ok || branch != "feature/feat-a" {
			t.Errorf("expected mismatch with feature/feat-a, got %s (ok=%v)", branch, ok)
		}
	})
}
