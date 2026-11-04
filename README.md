# zgt

English | [日本語](./README_ja.md)

A CLI tool that extends `git worktree add` by automatically copying ignored configuration files (like `.env`) to the new directory.

## Overview

Git's `worktree` feature is powerful,
but files ignored by `.gitignore` (such as `.env` or local configs) are not included in the newly created worktree. `zgt` automates the process of copying these files, allowing you to start development and testing immediately.

## Features

- **Standard Wrapper**: Works as a wrapper for `git worktree add`.
- **Auto-Discovery**: Automatically identifies and copies "ignored files" specified in `.gitignore`.
- **Structural Integrity**: Maintains directory structure during copy (e.g., config files inside `node_modules`).
- **Flexible Interface**: Powered by the Cobra framework for robust flag handling.
- **Path Automation**: Automatically generates worktree paths based on branch names (`{project}-{branch}`) adjacent to the repository root. Use `--path` for custom locations, `--base` to specify a base branch, and `--from-default` to force using the default branch as base.
- **Lifecycle Management**: Support for listing (`list`/`ls`) and removing (`remove`/`rm`) worktrees.
- **Port Management**: Automatically assigns unique port indexes to each worktree to prevent port collisions.
- **Custom Hooks**: Execute multiple shell commands naturally after creating (`add`) or removing (`rm`) worktrees.
- **Configuration Visibility**: `config` subcommand to inspect the final merged configuration and validate syntax.
- **Bi-directional Sync**: `sync` subcommand to synchronize ignored files from worktree back to the project root.
- **Agent Skills**: Distribute and install expert skills for AI agents.

## Badges

[![Release](https://img.shields.io/github/release/mocyuto/zgt.svg?style=for-the-badge)](https://github.com/mocyuto/zgt/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Build status](https://img.shields.io/github/actions/workflow/status/mocyuto/zgt/ci.yml?style=for-the-badge&branch=main)](https://github.com/mocyuto/zgt/actions?workflow=ci)

## Installation

### Homebrew

The easiest way to install on macOS is via Homebrew:

```bash
brew install mocyuto/tap/zgt
```

### Build

```bash
go build -o zgt .
```

### Move to PATH

Place the binary in a directory included in your `PATH`.

```bash
# Example for macOS / Linux
sudo mv zgt /usr/local/bin/zgt
```

## Development Workflow & Behavior

`zgt` is designed to make context switching seamless for developers by automating the repetitive parts of managing worktrees.

### 1. Starting a New Feature (`add`)

Running `git worktree add` usually leaves you with a fresh directory missing essential local files like `.env`. `zgt` automates the entire setup:

1. **Worktree Creation**: Creates the directory and checks out the branch.
2. **Auto-Path Generation**: Provide just the branch name, and it will be placed as `{project}-{branch}` at the same level as your repository root automatically. You can specify a custom path with `--path` or `-p`.
3. **Config Synchronization**: Identifies "ignored files" (like `.env`) in your main tree and copies them over, maintaining the directory structure.
4. **Port Reservation**: Reserves a unique "Port Index" for this specific worktree.
5. **Automated Setup**: If you define hooks like `npm install` in `hooks.add`, they run immediately after creation.

```bash
# Start a new feature in a fresh worktree (automated path)
zgt add feature-login

# Start a new feature from a specific base branch
zgt add feature-login --base develop

# Start a new feature forcing the default branch as base
zgt add feature-login --from-default

# Start a new feature in a custom directory
zgt add feature-login --path ./experimental-worktree
```

### 2. Running Multiple Projects Simultaneously (`env` / `ports`)

When running multiple servers across different worktrees, port collisions are a common pain point. `zgt` solves this:

- **Behavior**: Every worktree is assigned a stable index (`0, 1, 2...`).
- **Dynamic Calculation**: Port numbers are calculated by adding the index to the base port defined in the `zgt.config.yml` of the worktree's project root. This ensures correct calculation even with different base ports across projects.
- **Usage**: Define base ports (e.g., `api: 8080`) in your config. `zgt env` generates environment variables (e.g., `8080`, `8081`...) specific to that worktree.

```bash
# Move to a worktree and load its specific environment
cd ../my-project-feature-login
eval "$(zgt env)"

# $API_PORT is now 8081, preventing collision with other worktrees
npm start
```

### 3. Cleaning Up (`remove`)

When a feature is finished, `zgt` handles the teardown in one go.

- **Behavior**: Deletes the worktree directory and its associated local branch simultaneously (configurable).
- **Cleanup**: The Port Index is released and becomes available for future worktrees. You can also trigger cleanup scripts via `hooks.rm`.
- **Switched branches**: If you checked out a different branch inside a worktree at the auto-generated path (so the folder name no longer matches the branch), `zgt rm <original-branch>` asks whether to remove the folder too. Answering `y` removes the worktree folder; otherwise only the original branch is deleted. The branch currently checked out in the folder is kept.

```bash
# Done with the feature. Delete both directory and branch.
zgt rm feature-login
```

### 4. Synchronizing Ignored Files (`sync`)

If you've made changes to ignored configuration files (like `.env`) within your worktree and want to reflect those changes back to the main project root:

- **Interactive Mode**: Run `zgt sync` to open a TUI (powered by `rivo/tview`) where you can selectively choose which files to sync.
- **Bulk Sync**: Use `-a` / `--all` (or `--force`) to sync all ignored files immediately.
- **Path Filtering**: Use `-p` / `--path` to filter the list of files by a specific path or substring.
- **Exclude Filtering**: Use `-i` / `--ignore` to exclude files by path or substring (comma-separated).
- **Subdirectory-safe Paths**: When you run `zgt sync` from inside a subdirectory, synced files keep that relative path. For example, running from `src` and syncing `.env` updates `src/.env` in the main project root.

```bash
# Selectively sync changes back to root
zgt sync

# Sync files matching a specific path
zgt sync --path internal/config

# Sync files excluding certain paths
zgt sync --ignore "node_modules,temp"
```

### 5. Monitoring Assignments (`list` / `ports`)

- `list`: Shows which worktrees are active, their branch status, GitHub PR details, assigned ports, and whether they have uncommitted changes (`[DIRTY]`).
- `ports`: Shows the mapping of worktree paths to their assigned port indexes and actual port numbers. Use `-a` / `--all` to see assignments across all projects.

### 6. Other Commands

- `tmux ls`: Shows tmux sessions, windows, and panes status (Running/Waiting) in a hierarchical tree structure. Each pane also shows an `[agent: status]` badge when an opencode or Claude Code session is running inside it (see [AI Agent Status](#7-ai-agent-status-opencode--claude-code)).
- `tmux open`: Opens or activates the tmux window for the specified worktree. If the window exists, it switches to it; otherwise, it creates it according to the configuration. If no worktree name is provided, an interactive TUI is displayed to select one or more worktrees. Use `--profile <name>` to override (and persist) the profile used for window name and pane commands, mirroring `zgt add --profile`. In TUI mode the same profile is applied to all selected worktrees.
- `tmux close`: Gracefully closes the tmux window for the specified worktree (sends SIGTERM and waits).
- `ports update`: Synchronizes port assignments for the current project with the latest configuration. It adds missing port assignments and removes those no longer present in the configuration.
- `config`: Displays the final merged configuration (global + project + flags) in YAML format. Use `--check` to validate configuration syntax, or `--raw` to skip placeholder replacement.
- `config edit`: Edits the configuration file using the system editor. Defaults to editing the local project configuration.
- `version`: Prints the version number of `zgt`.
- `skill install`: Installs skills from the current repository's `skills/` directory to the global agent skills directory (`~/.claude/skills/`).
- `agent status`: Shows whether the opencode / Claude Code session in each worktree is `working`, `idle`, or `ask` (asking for permission/input). Use `-w` / `--watch` to refresh every 2s, and `-a` / `--all` to include worktrees with no active session.
- `agent install [claude|opencode|all]`: Installs the Claude Code hooks and opencode plugin that report session status to `zgt`. Run once per machine; re-running is safe.
- `agent uninstall [claude|opencode|all]`: Removes the hooks/plugin installed by `agent install`.

### 7. AI Agent Status (opencode / Claude Code)

`zgt` can show you, at a glance, whether the AI coding agent running inside a worktree is actively working, idle and awaiting input, or blocked asking for a permission/answer. This is surfaced in two places:

- `zgt tmux ls` appends a colored `[agent: status]` badge to each pane (green = working, cyan = ask, gray = idle).
- `zgt agent status` lists every worktree with its current agent status (color-coded) and age.

Status is written by hooks/plugins that `zgt agent install` sets up:

- **Claude Code**: command hooks for `UserPromptSubmit` → `working`, `Stop` → `idle`, `Notification` → `ask`, and `SessionEnd` → clear. The hooks call `zgt agent hook claude` which reads the event JSON from stdin.
- **opencode**: a plugin (`~/.config/opencode/plugins/zgt-status.js`) that maps session lifecycle and permission events to `working`/`idle`/`ask` (or clears the record on `session.deleted`). It accepts both `permission.asked` and `permission.updated` since opencode emits either depending on version. Late `message.updated` and late `permission.replied` events arriving within 500ms of `session.idle` are ignored so the status stays `idle`; a late `message.part.updated` (question completed/error) event is ignored whenever the session is already idle (regardless of timing, since plan/workflow answers often arrive after the debounce window); the next turn is detected primarily via `session.status:busy`/`retry`, with a user `message.updated` after the debounce window as a fallback.

```bash
# One-time setup: install hooks + plugin for both agents
zgt agent install

# See status for all worktrees
zgt agent status

# Live-updating view
zgt agent status --watch

# Install only one agent's integration
zgt agent install claude
zgt agent install opencode

# Remove when no longer needed
zgt agent uninstall
```

Status records live in `~/.config/zgt/agent-status/` keyed by worktree path and are pruned automatically after the configured `stale_after` duration (default `1h`), so crashed/killed sessions don't leave a stale "working" badge forever.

## Configuration

You can configure `zgt` globally (`~/.config/zgt/config.yaml`) or locally (`zgt.config.yml` in project root).

```yaml
# Examples of available configuration options
add:
  from_default: true # Always base new worktrees on the default branch (e.g., main)
  auto_pull: true # Pull updates on the default branch before creating a new worktree (automatically stashes uncommitted changes)

ignore:
  - .env
```

`zgt` loads configuration from three sources in this priority:

1. Local project configuration (`zgt.config.yaml` or `zgt.config.yml` in project root)
2. Global configuration (`~/.config/zgt/config.yaml`)
3. Explicit configuration path provided via `--config` flag

### Initializing Configuration (`init`)

You can generate a default configuration file (`zgt.config.yml`) in your project's root directory:

```bash
zgt init
```

This command performs the following:

1.  Creates a `zgt.config.yml` with sensible defaults for port management, environment templates, and sample hooks.
2.  Automatically appends `zgt.config.yml` and `zgt.config.yaml` to your `.gitignore` file to ensure they are not accidentally committed.

If `zgt.config.yml` or `zgt.config.yaml` already exists, the command will skip creation to prevent overwriting your existing settings.

### Customizing Configuration (`config edit`)

You can edit your configuration files directly from the CLI using your preferred editor (defined by `$EDITOR` or `vi`):

```bash
# Edit local configuration
zgt config edit --local

# Edit global configuration
zgt config edit --global
```

`zgt` will validate the YAML syntax before saving your changes.

### Project-Specific Configuration

You can create a `zgt.config.yaml` (or `.yml`) in your project's root directory to define settings specific to that project. Local settings for `hooks` and `ignore` will be **appended** to the global settings.

```yaml
# zgt.config.yaml
ignore:
  - "*.tmp"
  - "local-debug.log"

hooks:
  add:
    - "npm install"

git_hooks:
  enabled: true
  path: .githooks

ports:
  api: 8080
  web: 3000

tmux:
  enabled: true
  keep_open: true # Do not close the tmux window on 'rm' (default: false)
  panes:
    - id: main
      commands: ["yarn"]
    - id: dev
      target: main
      split: horizontal
      size: 50%
      commands: ["yarn dev"]

agent:
  enabled: true       # Show opencode / Claude Code status badges in `tmux ls` and `agent status` (default: true)
  stale_after: "1h"   # Discard status records older than this (default: "1h")
```

- `WEB_PORT=3001`

### Custom Environment Variables

You can define custom environment variables in the `env` section. These variables support placeholders and automatically exported via `zgt env`.

```yaml
env:
  COMPOSE_PROJECT_NAME: "zgt-{{.Repo}}"
  DEBUG: "true"
```

In your terminal:

```bash
eval "$(zgt env)"
echo $COMPOSE_PROJECT_NAME # zgt-myrepo
```

These variables are also available during [Custom Hooks](#custom-hooks) execution.

### Custom Ignore Patterns

You can specify additional file patterns to be ignored during the copy process. These patterns follow the same format as `.gitignore` (using `filepath.Match`).

```yaml
ignore:
  - ".env.production"
  - "secrets/*"
```

### Pull Default Branch After Removal

You can automatically pull the default branch from the remote repository after removing a worktree by adding a git command to your `rm` hooks.

```yaml
hooks:
  rm:
    - "git pull origin main:main"
```

### Git Hooks Registration

You can automatically register a Git hooks directory for each newly created worktree.

```yaml
git_hooks:
  enabled: true
  path: .githooks
  shared: true
```

- `path` supports absolute paths. Note that `zgt` always registers the resolved path as an **absolute path** in the Git configuration of the worktree, ensuring hooks work correctly regardless of the current working directory.
- `shared` (boolean):
  - `true` (default): Relative paths are resolved from the main project root. All worktrees share the same hooks directory.
  - `false`: The hooks directory is copied from the source repository to the new worktree. Each worktree uses its own independent copy of the hooks.
- If a worktree already has `core.hooksPath` configured, `zgt add` leaves it unchanged and prints a warning.

### Custom Hooks

Hooks allow you to run automated shell commands when worktrees are managed.

```yaml
hooks:
  # Commands to run after 'add'
  add:
    - "tmux new-window -n [{{.Repo}}]{{.Branch}} -c {{.Path}}"
    - "echo 'Welcome to {{.Repo}}'"
  # Commands to run after 'remove'
  rm:
    - "echo 'Cleanup for {{.Branch}}'"
```

#### Available Placeholders

Placeholders can be used in `hooks`, `env`, and `tmux.window_name` values.

| Placeholder          | Description                                                   |
| :------------------- | :------------------------------------------------------------ |
| `{{.Path}}`          | Absolute path of the worktree directory.                      |
| `{{.Repo}}`          | Name of the main project root directory.                      |
| `{{.CurrentDir}}`    | Name of the current working directory.                        |
| `{{.Branch}}`        | Name of the target branch (provided as argument).             |
| `{{.TargetBranch}}`  | Alias for `{{.Branch}}`.                                      |
| `{{.CurrentBranch}}` | Name of the branch you are currently on when executing `zgt`. |
| `{{.Profile}}`       | Selected profile name (empty for default). See [Profiles](#profiles). |

### Profiles

Profiles let you switch `env` overrides and **append** additional hook commands
per-worktree via `zgt add <branch> --profile <name>`. The selected profile is
persisted in `zgt state`, so `zgt env`, `zgt rm`, and the matching hooks all
continue to use the profile for the lifetime of that worktree.

A profile is defined under the top-level `profiles` map. Profile `env` keys
override top-level env per-key (top-level keys you do not mention are kept).
Profile `hooks.add` / `hooks.rm` are appended AFTER the top-level hooks,
so default behaviour remains unchanged and you only add extra steps.

```yaml
env:
  COMPOSE_PROJECT_NAME: "{{.Repo}}-{{.Branch}}"
  DB_HOST: "db"

profiles:
  migration:
    env:
      # Override DB_HOST only; COMPOSE_PROJECT_NAME is inherited from top-level.
      DB_HOST: "iso-db"
    hooks:
      add:
        # Run AFTER the top-level `docker compose up -d api envoy` etc.
        - "./scripts/db-clone.sh"
      rm:
        # Run AFTER the top-level `docker compose down`.
        - "docker compose down -v"
```

Usage:

```bash
# Normal worktree (uses shared DB):
zgt add feat/foo

# Worktree with isolated DB clone (DB volume copied on `add`, wiped on `rm`):
zgt add feat/migrate-bar --profile migration
```

An empty or `--profile default` value selects the implicit default profile
(top-level only), which keeps behaviour fully backward compatible. Unknown
profile names are rejected at `zgt add` time. See `example/` for a complete
`docker-compose.yaml + zgt.config.yaml + scripts/db-clone.sh` walkthrough that
implements the "main worktree owns the shared DB, migration worktrees clone
its Docker volume" pattern.

#### Profile `tmux` override

A profile may also overlay a `tmux` section onto the top-level `tmux` config.
The merge is per-field:

- `enabled` / `keep_open`: OR semantics — the profile can turn the flag **on**,
  but never off. So a profile can opt into tmux even if the top-level left it
  disabled, and a profile can keep the window open on `zgt rm` regardless of
  the global default.
- `window_name`: overridden when the profile sets a non-empty value. This is
  the common lever for distinguishing profile windows in the tmux window list
  (e.g. `[repo]fe-branch` vs `[repo]branch`).
- `panes`: replaced wholesale when the profile defines at least one pane;
  otherwise the top-level panes are used as-is.

```yaml
tmux:
  enabled: true
  window_name: "[{{.Repo}}]{{.Branch}}"
  panes:
    - id: main
      commands: ["yarn"]

profiles:
  frontend:
    tmux:
      keep_open: true
      window_name: "[{{.Repo}}]fe-{{.Branch}}"
      panes:
        - id: fe
          commands: ["npm run dev"]
```

The profile-resolved `tmux` config drives `zgt add`, `zgt tmux open`, and
`zgt tmux close` (the close command now also resolves the profile from
`zgt state`, matching `zgt rm`). Note that `zgt tmux close` force-closes
the window regardless of `keep_open`, so a profile that turned `keep_open`
on will still keep the window alive across `zgt rm` but will close on an
explicit `zgt tmux close`.

The same per-field rules apply when a profile is defined in **both** the
global and local config: `enabled`/`keep_open` are OR-ed across the two
profile definitions, `window_name` is overridden by the local profile when
non-empty, and `panes` are replaced by whichever side defines any (local
replaces global; otherwise global wins).

#### Template Functions

You can use functions to transform placeholder values.

| Function   | Description                                              | Example                                |
| :--------- | :------------------------------------------------------- | :------------------------------------- |
| `hostname` | Replaces hostname-unsafe characters (`_`, `/`) with `-`. | `{{.Branch \| hostname}}`->`feat-test` |

### Tmux Integration

`zgt` can automatically set up a tmux window with multiple panes and execute commands in each when you run `add`. You can explicitly target panes for splitting using IDs.

```yaml
tmux:
  enabled: true
  panes:
    - id: main
      commands: ["yarn"]
    - id: side
      target: main
      split: horizontal
      size: 50%
      commands: ["yarn dev"]
    - target: side
      split: vertical
      commands: ["yarn watch"]
    - target: main
      split: vertical
      commands: ["tail -f logs/app.log"]
```

#### Pane Properties

| Property      | Description                                                                   |
| :------------ | :---------------------------------------------------------------------------- |
| `enabled`     | Whether to enable tmux integration (`true` / `false`).                        |
| `keep_open`   | Whether to NOT close the window when running `zgt rm` (default: `false`).     |
| `window_name` | (Optional) Template for the tmux window name.                                 |
| `id`          | (Optional) Unique ID for the pane to be referenced as a `target`.             |
| `target`      | (Optional) ID of the pane to split. If omitted, splits the last created pane. |
| `commands`    | List of commands to execute in the pane.                                      |
| `split`       | Split direction: `horizontal` (h) or `vertical` (v).                          |
| `size`        | Pane size (e.g., `20%` for percentage or `20` for lines/columns).             |

If `enabled` is `true`, `zgt` will:

1. Create a new tmux window named as specified in `window_name` (defaults to `[repo]branch`).
2. Follow the `panes` list to create splits. Each split targets the specified `target` or the last created pane.
3. Execute the `commands` in each pane and keep the shell open.

#### Note

- Requires `tmux` to be installed and a tmux session to be running.

#### Note

- If you only need a single command, you can use a string instead of a list: `add: "echo hello"`.
- Commands are executed via `/bin/sh -c`, allowing for pipes and status checks.

## Agent Skill Management

`zgt` provides a way to manage "Expert Skills" for AI agents. Skills are sets of instructions and resources that extend an agent's capabilities.

### Installing Skills (`skill install`)

To install the skills from the current repository so that your AI collaborator (like Claude) can use them:

```bash
zgt skill install
```

This will open an interactive TUI to choose from 4 installation targets:

- **Local .claude**: `./.claude/skills/`
- **Local .agents**: `./.agents/skills/`
- **Global .claude**: `~/.claude/skills/`
- **Global .agents**: `~/.agents/skills/`

Use `-a` or `--all` to install to all targets immediately.

## Requirements

- `git` and `gh` must be installed and available in your environment.

## Development & Testing

Toolchain versions (Go and Bun) are managed via [`mise.toml`](./mise.toml). Run `mise install` once to set them up, then use the mise tasks:

```bash
mise install
mise run test       # Go tests: go test ./...
mise run test-js    # JS tests: bun test internal/agentstatus/assets/
mise run test-all  # both, in parallel
```

You can also invoke the underlying commands directly:

```bash
go test -v ./...
bun test internal/agentstatus/assets/
```

The opencode status plugin (`internal/agentstatus/assets/zgt-status.js`) has its own State-transition tests under the same directory; run them with `mise run test-js` whenever you edit the plugin.

## FAQ

### Q: What happens if I `add` a branch that already exists?

`zgt` will check if the branch exists locally or on the remote (`origin`).

- If it exists, `zgt` will use the existing branch for the new worktree.
- Note that Git does not allow the same branch to be checked out in multiple worktrees simultaneously. If the branch is already in use elsewhere, the command will fail with a Git error.

### Q: Does the `rm` command delete the created directory?

Yes, `zgt rm` (or `remove`) will:

1. Delete the worktree directory from your filesystem.
2. Unregister the worktree from Git.
3. Delete the associated local branch (unless `--keep-branch` is used).
   If there are uncommitted changes, Git will protect the directory from deletion unless `-f` (`--force`) is used.
