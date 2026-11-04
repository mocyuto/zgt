---
name: zgt-usage
description: Instructions for AI agents on how to use `zgt` to manage Git worktrees and development environments effectively.
---

# ZGT Usage Skill

This skill provides instructions for AI agents on how to use `zgt` to manage Git worktrees and development environments effectively.

## Overview

`zgt` is a CLI tool that extends `git worktree add` by automatically copying ignored configuration files (like `.env`) to the new directory. It also manages ports and tmux sessions.

For detailed configuration options, see the [Configuration Guide](./config.md).

## Core Commands

### 1. Adding a Worktree (`add`)

Use `zgt add <branch>` to create a new worktree.

- It automatically generates a path: `../{project}-{branch}`.
- Use `--path <dir>` to specify a custom target directory.
- Use `--base <branch>` (or `-b`) to specify a base branch to create the worktree from.
- Use `--from-default` to force using the default branch as base.
- It copies ignored files specified in `.gitignore` or `zgt.config.yml`.
- It registers Git hooks if `git_hooks` is enabled (shares or copies based on `shared` setting).
- It executes `hooks.add` (e.g., `npm install`).
- **Existing Branch**: If the branch already exists, it uses it. Note that a branch cannot be checked out in multiple worktrees at once.

### 2. Removing a Worktree (`remove` / `rm`)

Use `zgt rm <branch>` to cleanup.

- It deletes the worktree directory from your filesystem.
- It deletes the associated local branch (controllable via `--keep-branch`).
- It safely closes the associated tmux window (if `tmux.keep_open` is false).
- It releases the assigned port index.
- If a different branch was checked out inside a worktree (no longer matches the branch it was created with), `zgt rm <original-branch>` prompts `[y/N]` whether to remove the folder too. `y` removes the worktree folder; otherwise only the original branch is deleted. The currently checked-out branch is kept. Pipe `y`/`n` (e.g. `echo y | zgt rm <branch>`) when running non-interactively; with no input it aborts without deleting anything. The original branch is recorded at `zgt add` time and shown in `zgt rm` completion; `{{.Branch}}` / `{{.TargetBranch}}` in hooks keep resolving to it, while `{{.CurrentBranch}}` reflects the branch where `zgt` was run.

- Use `zgt sync` to bring changes from a worktree back to the project root.
- Use `zgt sync -a` / `--all` for immediate sync of all files.
- Use `zgt sync -p <string>` / `--path <string>` to filter files to include.
- Use `zgt sync -i <string>` / `--ignore <string>` to filter files to exclude (comma-separated).
- If `zgt sync` is run from a subdirectory, synced files preserve that relative path under the main project root.
- Without flags, it opens an interactive TUI.

### 4. Environment and Ports (`env` / `ports` / `tmux`)

- `zgt env`: Outputs export commands for environment variables (eval "$(zgt env)").
- `zgt ports`: Shows port assignments.
- `zgt tmux open [branch]`: Opens or activates the tmux window for a worktree. Opens an interactive TUI if no branch is specified. Use `--profile <name>` to override (and persist) the profile used for window name and pane commands, mirroring `zgt add --profile`. In TUI mode the same profile is applied to all selected worktrees.
- `zgt tmux close <branch>`: Gracefully closes the tmux window for a worktree.
- `zgt tmux ls`: Lists tmux sessions/windows/panes with Running/Waiting status and an `[agent: status]` badge for any pane running an opencode / Claude Code session (green=working, cyan=ask, gray=idle).

### 4b. AI Agent Status (`agent`)

`zgt` reports whether the opencode or Claude Code session in each worktree is `working`, `idle`, or `ask` (asking for permission/input).

- `zgt agent install [claude|opencode|all]`: One-time setup. Installs Claude Code hooks (`~/.claude/settings.json`) and the opencode plugin (`~/.config/opencode/plugins/zgt-status.js`) that report session status to `zgt`. The opencode plugin maps `session.idle`/`session.status`(idle)/`session.created`/`session.error`→`idle`, `permission.asked`/`permission.updated` and the built-in `question` tool (via `message.part.updated`)→`ask`, `permission.replied`/`question`-completed/`session.status`(busy/retry)→`working`, `session.deleted`→clear; on plugin init (agent restart) it writes `idle` so stale status from a crashed session doesn't linger; late `message.updated` and late `permission.replied` events within 500ms of `session.idle` are ignored, and a late `message.part.updated` (question completed/error) is ignored whenever the session is already idle (regardless of timing, so a plan/workflow answer arriving after the debounce window doesn't flip idle back to working) — the next turn is detected via `session.status:busy`/`retry`, with a user `message.updated` after the debounce window as a fallback. Re-running is safe and idempotent.
- `zgt agent status [worktree]`: Lists every worktree with its current agent status and age. Use `-w`/`--watch` for a live 2s-refreshing view, and `-a`/`--all` to include worktrees with no active session.
- `zgt agent uninstall [claude|opencode|all]`: Removes the hooks/plugin.

Status records are keyed by worktree path under `~/.config/zgt/agent-status/` and auto-pruned after `agent.stale_after` (default `1h`). Disable the integration in `zgt.config.yml` with `agent.enabled: false`.

### 5. Global Flags

- `--verbose` / `-v`: Shows detailed output and prints the current merged configuration at the start of any command execution.

## Best Practices for Agents

1. **New Feature Development**: When starting a new task, always check if a worktree is needed. Use `zgt add` to set up a clean environment.
2. **Path Context**: After `zgt add`, remember that the project root is sibling to your current repository.
3. **Configuration**: If the user needs specific files copied, suggest adding them to the `ignore` list in `zgt.config.yml`.
4. **Cleanup**: Proactively suggest `zgt rm` when a task is completed and the PR is merged.
5. **Hostname Compatibility**: If using placeholders for hostnames (e.g., in tmux window names or environment variables), use the `hostname` function to replace illegal characters: `{{.Branch | hostname}}`.
