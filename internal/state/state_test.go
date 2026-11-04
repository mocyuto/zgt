package state

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAssignPortIndex(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "zgt-assign-test")
	defer os.RemoveAll(tmpDir)

	// Initialize git in tmpDir to make it a "main" project
	exec.Command("git", "init", tmpDir).Run()
	absMain, _ := filepath.Abs(tmpDir)
	projectName := filepath.Base(absMain)

	AppState.Projects = make(map[string]ProjectState)

	// Assign to main (should get 0)
	idx_main := AssignPortIndex(projectName, absMain, "http", 3000)
	if idx_main != 0 {
		t.Errorf("Expected index 0 for main project, got %d", idx_main)
	}

	// Assign to a "worktree" (not the main root)
	wtPath := filepath.Join(tmpDir, "wt0")
	os.MkdirAll(wtPath, 0755)
	idx_wt := AssignPortIndex(projectName, wtPath, "http", 3000)
	if idx_wt != 1 {
		t.Errorf("Expected index 1 for worktree, got %d", idx_wt)
	}

	// Assign another worktree
	wtPath2 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wtPath2, 0755)
	idx_wt2 := AssignPortIndex(projectName, wtPath2, "http", 3000)
	if idx_wt2 != 2 {
		t.Errorf("Expected index 2 for second worktree, got %d", idx_wt2)
	}

	// Re-assign existing path
	idx_wt_again := AssignPortIndex(projectName, wtPath, "http", 3000)
	if idx_wt_again != 1 {
		t.Errorf("Expected index 1 for same worktree, got %d", idx_wt_again)
	}

	// Different project should be able to use index 0 as well (project-scoped)
	projectName2 := "other-project"
	idx_pj2 := AssignPortIndex(projectName2, "/other/path", "http", 3000)
	// it starts at 1 for non-main path
	if idx_pj2 != 1 {
		t.Errorf("Expected index 1 for non-main path in other project, got %d", idx_pj2)
	}
}

func TestCleanupState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zgt-state-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	existingPath := filepath.Join(tmpDir, "exists")
	os.MkdirAll(existingPath, 0755)
	nonExistingPath := filepath.Join(tmpDir, "missing")

	AppState.Projects = map[string]ProjectState{
		"pj": {
			Worktrees: map[string]*WorktreeState{
				existingPath:    {Ports: map[string]int{"http": 0}},
				nonExistingPath: {Ports: map[string]int{"http": 1}},
			},
		},
	}

	CleanupState()

	if _, ok := AppState.Projects["pj"].Worktrees[existingPath]; !ok {
		t.Error("Existing path was incorrectly removed")
	}
	if _, ok := AppState.Projects["pj"].Worktrees[nonExistingPath]; ok {
		t.Error("Non-existing path was not removed")
	}
}

func TestGetCurrentWorktreePorts(t *testing.T) {
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	tmpDir, err := os.MkdirTemp("", "zgt-match-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	wtPath := filepath.Join(tmpDir, "worktree")
	subDir := filepath.Join(wtPath, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	wtPath, _ = filepath.EvalSymlinks(wtPath)
	subDir, _ = filepath.EvalSymlinks(subDir)

	AppState.Projects = map[string]ProjectState{
		"pj": {
			Worktrees: map[string]*WorktreeState{
				wtPath: {Ports: map[string]int{"http": 5}},
			},
		},
	}

	// Test exact match
	if err := os.Chdir(wtPath); err != nil {
		t.Fatal(err)
	}
	ports, found := GetCurrentWorktreePorts()
	if !found || ports["http"] != 5 {
		t.Errorf("Expected {http:5}, got %v", ports)
	}

	// Test subdirectory match
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	ports, found = GetCurrentWorktreePorts()
	if !found || ports["http"] != 5 {
		t.Errorf("Expected {http:5} for subdirectory, got %v", ports)
	}
}

func TestSaveLoadState(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "zgt-home-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Setup state
	AppState.Projects = map[string]ProjectState{
		"pj1": {
			Worktrees: map[string]*WorktreeState{
				"/abs/path/1": {Ports: map[string]int{"http": 10}},
			},
		},
	}

	if err := SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Clear and load
	AppState.Projects = nil
	if err := LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if AppState.Projects["pj1"].Worktrees["/abs/path/1"].Ports["http"] != 10 {
		t.Errorf("Loaded project state mismatch: %v", AppState.Projects)
	}
}

func TestSetProfile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "zgt-set-profile-test")
	defer os.RemoveAll(tmpDir)
	exec.Command("git", "init", tmpDir).Run()

	absMain, _ := filepath.Abs(tmpDir)
	projectName := filepath.Base(absMain)
	wtPath := filepath.Join(tmpDir, "wt")
	os.MkdirAll(wtPath, 0755)

	AppState.Projects = make(map[string]ProjectState)

	// Set profile for a new worktree
	SetProfile(projectName, wtPath, "migration")

	// Existing ports should be preserved when SetProfile is called on a
	// worktree that already has port assignments (simulated here by assigning
	// first and then setting profile).
	AssignPortIndex(projectName, wtPath, "http", 3000)
	SetProfile(projectName, wtPath, "migration")

	proj, ok := AppState.Projects[projectName]
	if !ok {
		t.Fatalf("project not found")
	}
	wt, ok := proj.Worktrees[NormalizePath(wtPath)]
	if !ok {
		t.Fatalf("worktree not found")
	}
	if wt.Profile != "migration" {
		t.Errorf("Profile = %q, want %q", wt.Profile, "migration")
	}
	if wt.Ports["http"] != 1 {
		t.Errorf("Ports[http] = %d, want 1 (preserved)", wt.Ports["http"])
	}
}

func TestGetCurrentWorktree_Profile(t *testing.T) {
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	tmpDir, _ := os.MkdirTemp("", "zgt-profile-match")
	defer os.RemoveAll(tmpDir)
	wtPath := filepath.Join(tmpDir, "wt")
	os.MkdirAll(wtPath, 0755)
	wtPath, _ = filepath.EvalSymlinks(wtPath)

	AppState.Projects = map[string]ProjectState{
		"pj": {
			Worktrees: map[string]*WorktreeState{
				wtPath: {Ports: map[string]int{"http": 3}, Profile: "migration"},
			},
		},
	}

	if err := os.Chdir(wtPath); err != nil {
		t.Fatal(err)
	}

	wt, ok := GetCurrentWorktree()
	if !ok {
		t.Fatalf("GetCurrentWorktree returned ok=false")
	}
	if wt.Profile != "migration" {
		t.Errorf("Profile = %q, want %q", wt.Profile, "migration")
	}
	if wt.Ports["http"] != 3 {
		t.Errorf("Ports[http] = %d, want 3", wt.Ports["http"])
	}

	ports, ok := GetCurrentWorktreePorts()
	if !ok || ports["http"] != 3 {
		t.Errorf("GetCurrentWorktreePorts = %v (ok=%v)", ports, ok)
	}
}

func TestSetBranchPreservesOtherFields(t *testing.T) {
	tmpDir, _ := filepath.EvalSymlinks(t.TempDir())
	wtPath := filepath.Join(tmpDir, "wt")
	os.MkdirAll(wtPath, 0755)

	AppState.Projects = make(map[string]ProjectState)
	AssignPortIndex("pj", wtPath, "http", 3000)
	SetProfile("pj", wtPath, "migration")
	SetBranch("pj", wtPath, "hoge")

	wt := AppState.Projects["pj"].Worktrees[NormalizePath(wtPath)]
	if wt.Branch != "hoge" || wt.Profile != "migration" || wt.Ports["http"] != 1 {
		t.Errorf("unexpected state: %+v", wt)
	}
}

func TestFindWorktreeAndOriginalBranch(t *testing.T) {
	tmpDir, _ := filepath.EvalSymlinks(t.TempDir())
	outerPath := filepath.Join(tmpDir, "repo-hoge")
	nestedPath := filepath.Join(outerPath, ".worktrees", "x")
	os.MkdirAll(filepath.Join(nestedPath, "sub"), 0755)

	AppState.Projects = map[string]ProjectState{
		"repo": {Worktrees: map[string]*WorktreeState{
			outerPath: {Branch: "hoge"},
		}},
	}

	if got := OriginalBranch(outerPath); got != "hoge" {
		t.Errorf("OriginalBranch(outer) = %q, want hoge", got)
	}
	// 管理外のネストworktreeは親のブランチを継承しない
	if got := OriginalBranch(nestedPath); got != "" {
		t.Errorf("OriginalBranch(nested) = %q, want empty", got)
	}
	// FindWorktreeはcwd用にサブディレクトリも一致
	if wt, ok := FindWorktree(filepath.Join(nestedPath, "sub")); !ok || wt.Branch != "hoge" {
		t.Errorf("FindWorktree(nested/sub) = %+v, %v", wt, ok)
	}
	// 前方一致だけのパスは対象外
	if _, ok := FindWorktree(outerPath + "-x"); ok {
		t.Error("FindWorktree(repo-hoge-x) should not match repo-hoge")
	}
}

func TestFindPathsByBranch(t *testing.T) {
	tmpDir, _ := filepath.EvalSymlinks(t.TempDir())
	a := filepath.Join(tmpDir, "a")
	b := filepath.Join(tmpDir, "b")
	gone := filepath.Join(tmpDir, "gone")
	os.MkdirAll(a, 0755)
	os.MkdirAll(b, 0755)

	AppState.Projects = map[string]ProjectState{
		"pj": {Worktrees: map[string]*WorktreeState{
			b:                          {Branch: "hoge"},
			a:                          {Branch: "hoge"},
			gone:                       {Branch: "hoge"},
			filepath.Join(tmpDir, "c"): {},
		}},
	}
	got := FindPathsByBranch("pj", "hoge")
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Errorf("FindPathsByBranch(hoge) = %v, want [%s %s]", got, a, b)
	}
	for _, c := range []struct{ pj, br string }{{"pj", "fuga"}, {"pj", ""}, {"other", "hoge"}} {
		if got := FindPathsByBranch(c.pj, c.br); len(got) != 0 {
			t.Errorf("FindPathsByBranch(%q, %q) = %v, want empty", c.pj, c.br, got)
		}
	}
}

func TestClearBranch(t *testing.T) {
	AppState.Projects = map[string]ProjectState{
		"pj": {Worktrees: map[string]*WorktreeState{"/a": {Branch: "hoge", Profile: "p"}}},
	}
	ClearBranch("pj", "/a")
	if wt := AppState.Projects["pj"].Worktrees["/a"]; wt.Branch != "" || wt.Profile != "p" {
		t.Errorf("unexpected state after ClearBranch: %+v", wt)
	}
	// 未登録パス・プロジェクトはエントリを作らない
	ClearBranch("pj", "/missing")
	ClearBranch("none", "/a")
	if len(AppState.Projects["pj"].Worktrees) != 1 || len(AppState.Projects) != 1 {
		t.Errorf("ClearBranch should not create entries: %+v", AppState.Projects)
	}
}
