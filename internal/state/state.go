package state

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func NormalizePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return eval
}

type State struct {
	Projects map[string]ProjectState `json:"projects"` // map[projectName]ProjectState
}

type ProjectState struct {
	Worktrees map[string]*WorktreeState `json:"worktrees"` // map[absPath]*WorktreeState
}

type WorktreeState struct {
	Ports   map[string]int `json:"ports"`             // map[portKey]portIndex
	Profile string         `json:"profile,omitempty"` // selected profile name (empty for default)
	Branch  string         `json:"branch,omitempty"`  // branch name at `zgt add` time
}

var AppState State

func getStateFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zgt", "state.json"), nil
}

func LoadState() error {
	path, err := getStateFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		AppState.Projects = make(map[string]ProjectState)
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &AppState)
	if err != nil {
		// If unmarshal fails (e.g. old format), reset state
		AppState.Projects = make(map[string]ProjectState)
		return nil
	}
	if AppState.Projects == nil {
		AppState.Projects = make(map[string]ProjectState)
	}
	return nil
}

func SaveState() error {
	path, err := getStateFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(AppState, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func AssignPortIndex(projectName, path, portKey string, basePort int) int {
	path = NormalizePath(path)
	if AppState.Projects == nil {
		AppState.Projects = make(map[string]ProjectState)
	}

	proj, ok := AppState.Projects[projectName]
	if !ok {
		proj = ProjectState{
			Worktrees: make(map[string]*WorktreeState),
		}
	}

	wt, ok := proj.Worktrees[path]
	if !ok {
		wt = &WorktreeState{Ports: make(map[string]int)}
	}

	if idx, ok := wt.Ports[portKey]; ok {
		return idx
	}

	// Determine if this is the main project path (Git root)
	// In the main worktree, .git is a directory.
	// In linked worktrees, .git is a file.
	isMain := false
	gitPath := filepath.Join(path, ".git")
	if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
		isMain = true
	}

	// Find the smallest available index within THIS project
	usedInProject := make(map[int]bool)
	for _, wState := range proj.Worktrees {
		if idx, found := wState.Ports[portKey]; found {
			usedInProject[idx] = true
		}
	}

	idx := 1
	if isMain {
		idx = 0
	}

	for {
		if !usedInProject[idx] {
			if isPortAvailable(basePort + idx) {
				break
			}
			usedInProject[idx] = true
		}
		idx++
	}

	wt.Ports[portKey] = idx
	proj.Worktrees[path] = wt
	AppState.Projects[projectName] = proj
	return idx
}

func isPortAvailable(port int) bool {
	// Simple check using net.Listen
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func ReleasePortIndex(projectName, path string) {
	path = NormalizePath(path)
	if proj, ok := AppState.Projects[projectName]; ok {
		delete(proj.Worktrees, path)
		if len(proj.Worktrees) == 0 {
			delete(AppState.Projects, projectName)
		} else {
			AppState.Projects[projectName] = proj
		}
	}
}

func CleanupState() {
	// Remove paths that no longer exist
	for projectName, proj := range AppState.Projects {
		changed := false
		for path := range proj.Worktrees {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				delete(proj.Worktrees, path)
				changed = true
			}
		}
		if changed {
			if len(proj.Worktrees) == 0 {
				delete(AppState.Projects, projectName)
			} else {
				AppState.Projects[projectName] = proj
			}
		}
	}
}

func GetCurrentWorktreePorts() (map[string]int, bool) {
	wt, ok := GetCurrentWorktree()
	if !ok {
		return nil, false
	}
	return wt.Ports, true
}

// GetWorktreePortsByPath returns the port-key->idx map for the worktree at the
// given path (normalized internally), or (nil, false) if not found. Useful
// for path-keyed lookups (e.g. injecting port vars into hook subprocesses
// whose ctx.Path is the worktree even though zgt itself was invoked from
// elsewhere).
func GetWorktreePortsByPath(path string) (map[string]int, bool) {
	n := NormalizePath(path)
	for _, proj := range AppState.Projects {
		if wt, ok := proj.Worktrees[n]; ok {
			return wt.Ports, true
		}
	}
	return nil, false
}

// GetCurrentWorktree returns the WorktreeState for the worktree containing the
// current working directory, or (nil, false) if not found.
func GetCurrentWorktree() (*WorktreeState, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, false
	}
	return FindWorktree(cwd)
}

// FindWorktree returns the WorktreeState for the most specific managed
// worktree containing path (path itself or any subdirectory of it).
func FindWorktree(path string) (*WorktreeState, bool) {
	absPath := NormalizePath(path)

	var best *WorktreeState
	bestLen := -1
	for _, proj := range AppState.Projects {
		for p, wt := range proj.Worktrees {
			rel, err := filepath.Rel(p, absPath)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				continue
			}
			if len(p) > bestLen {
				best, bestLen = wt, len(p)
			}
		}
	}
	return best, best != nil
}

// OriginalBranch returns the branch recorded at `zgt add` time for the
// worktree rooted exactly at path, or "" if unknown. State must be loaded.
func OriginalBranch(path string) string {
	n := NormalizePath(path)
	for _, proj := range AppState.Projects {
		if wt, ok := proj.Worktrees[n]; ok {
			return wt.Branch
		}
	}
	return ""
}

// FindPathsByBranch returns existing worktree paths (sorted) whose recorded
// original branch equals branch within the given project.
func FindPathsByBranch(projectName, branch string) []string {
	if branch == "" {
		return nil
	}
	var res []string
	for p, wt := range AppState.Projects[projectName].Worktrees {
		if wt.Branch != branch {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue // 手動削除済みのstaleエントリ
		}
		res = append(res, p)
	}
	sort.Strings(res)
	return res
}

// SetBranch records the original branch for the given worktree path within a
// project. The caller is responsible for persisting it via SaveState.
func SetBranch(projectName, path, branch string) {
	getOrCreate(projectName, path).Branch = branch
}

// ClearBranch removes the recorded original branch if the worktree entry
// exists. The caller is responsible for persisting it via SaveState.
func ClearBranch(projectName, path string) {
	if wt, ok := AppState.Projects[projectName].Worktrees[NormalizePath(path)]; ok {
		wt.Branch = ""
	}
}

// SetProfile stores the profile name for the given worktree path within a
// project. The state is updated in memory; the caller is responsible for
// persisting it via SaveState.
func SetProfile(projectName, path, profile string) {
	getOrCreate(projectName, path).Profile = profile
}

// プロジェクト・worktreeエントリの取得（なければ作成）
func getOrCreate(projectName, path string) *WorktreeState {
	path = NormalizePath(path)
	if AppState.Projects == nil {
		AppState.Projects = make(map[string]ProjectState)
	}
	proj, ok := AppState.Projects[projectName]
	if !ok {
		proj = ProjectState{Worktrees: make(map[string]*WorktreeState)}
	}
	wt, ok := proj.Worktrees[path]
	if !ok {
		wt = &WorktreeState{Ports: make(map[string]int)}
	}
	proj.Worktrees[path] = wt
	AppState.Projects[projectName] = proj
	return wt
}
