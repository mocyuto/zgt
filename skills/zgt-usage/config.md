# ZGT Configuration Guide

`zgt` can be configured globally or per-project using YAML files.

## Configuration Files

`zgt` loads configuration files in the following order of priority:

1. **Local Project Config**: `zgt.config.yml` or `zgt.config.yaml` in the project root.
2. **Global Config**: `~/.config/zgt/config.yaml`.
3. **Explicit Config**: Path specified via the `--config` flag.

### Generating Configuration (`init`)

Run `zgt init` in your project root to generate a default `zgt.config.yml` with port management, environment templates, and sample hooks. It also adds the config files to `.gitignore`.

### Editing Configuration (`config edit`)

You can edit configuration files directly using your system editor:

- `zgt config edit --local`: Edit the local project configuration.
- `zgt config edit --global`: Edit the global configuration.

The CLI automatically performs a YAML syntax check before saving.

## Configuration Options

### `add`

Options for the `add` command.

- `from_default` (bool): If true, always create new worktrees from the default branch (e.g., `main`).
- `auto_pull` (bool): If true, pull the latest changes from the remote default branch before creating a worktree.

### `ignore`

A list of file patterns to be copied to the new worktree even if they are ignored by Git (e.g., `.env`).

```yaml
ignore:
  - ".env"
  - "config/*.local.json"
```

### `hooks`

Custom shell commands to execute during worktree lifecycle.

- `add`: Commands run after `zgt add`.
- `rm`: Commands run after `zgt rm`.

### `git_hooks`

Automatically configure Git hooks for newly created worktrees.

- `enabled` (bool): If true, `zgt add` sets `core.hooksPath` in the new worktree.
- `path` (string): Path to the hooks directory. Default is `.githooks`. `zgt` registers this as an **absolute path** in the Git configuration.
- `shared` (bool): If true (default), relative paths are resolved from the main project root and shared. If false, the hooks directory is copied to the new worktree.

### `ports`

Base port assignments for the project. `zgt` uses these to calculate unique ports for each worktree based on its index.

```yaml
ports:
  api: 8080
  web: 3000
```

### `env`

Custom environment variables that can include placeholders. These are exported via `zgt env`.

```yaml
env:
  COMPOSE_PROJECT_NAME: "zgt-{{.Repo}}"
```

### `tmux`

Tmux integration settings.

- `enabled` (bool): Enable automatic tmux window/pane creation.
- `keep_open` (bool): If true, do not close the tmux window on `zgt rm`. Default is `false` (closes by default).
- `window_name` (string): Template for the tmux window name.
- `panes` (list): List of pane configurations.
  - `id`: Unique identifier for the pane.
  - `target`: ID of the pane to split.
  - `split`: `horizontal` or `vertical`.
  - `size`: Percentage or number of lines/columns.
  - `commands`: List of commands to run in the pane.

### `profiles`

A map of named profiles selectable with `zgt add <branch> --profile <name>`.
The selected profile is stored in zgt's state for that worktree, so `zgt env`,
`zgt rm`, and hooks continue to use the profile for the lifetime of the
worktree. Each profile value supports `env` (per-key override of top-level
env), `hooks` (the `add`/`rm` slices are appended AFTER the top-level
hooks), and `tmux` (per-field overlay onto the top-level `tmux`).

```yaml
profiles:
  migration:
    env:
      DB_HOST: "iso-db"
    hooks:
      add:
        - "./scripts/db-clone.sh"
      rm:
        - "docker compose down -v"
```

An empty or `default` profile name matches the implicit default behavior
(top-level only), so the `profiles` key is fully optional. The `{{.Profile}}`
placeholder exposes the active profile name to `hooks`/`env` templates. The
`example/` directory in the repo demonstrates the "isolated DB per migration
worktree" pattern end-to-end.

Profile `tmux` overlay merges per-field onto the top-level `tmux`:

- `enabled` / `keep_open`: OR semantics (profile can enable, never disable).
- `window_name`: overridden when the profile sets a non-empty value.
- `panes`: replaced wholesale when the profile defines at least one pane;
  otherwise the top-level panes are inherited.

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
`zgt tmux close` consistently. `zgt tmux close` force-closes regardless of
`keep_open`. The same per-field rules apply when a profile is defined in
both global and local config (local overrides global for `window_name`,
replaces `panes`, and ORs `enabled`/`keep_open`).

## Placeholders

Placeholders can be used in `hooks`, `env`, and `tmux.window_name`.

| Placeholder          | Description                              |
| :------------------- | :--------------------------------------- |
| `{{.Path}}`          | Absolute path of the worktree directory. |
| `{{.Repo}}`          | Name of the main project root directory. |
| `{{.CurrentDir}}`    | Current working directory name.          |
| `{{.Branch}}`        | Target branch name. Stays the branch used at `zgt add` even after switching branches inside the worktree. |
| `{{.TargetBranch}}`  | Alias for `{{.Branch}}`.                 |
| `{{.CurrentBranch}}` | Branch name where `zgt` was executed.    |
| `{{.Profile}}`       | Selected profile name (empty for default). |

## Inspecting Configuration (`config`)

Use `zgt config` to view the final merged configuration.

- `--check`: Validate configuration for errors.
- `--raw`: Display configuration without placeholder replacement.
