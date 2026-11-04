# zgt

`git worktree` を作成する際、設定ファイル（`.env` など）を自動的にコピーして新しいディレクトリを作成する CLI ツールです。

## 概要

Git の `worktree` 機能は便利ですが、`.gitignore` で除外されている `.env` やローカルの設定ファイルなどは、新しく作成したワークツリーには含まれません。`zgt` を使うことで、これらを自動的にコピーし、すぐに開発やテストが可能なワークツリーを作成できます。

## 特徴

- `git worktree add` のラッパーとして動作
- `.gitignore` に指定された「無視されているファイル」を自動的に特定してコピー
- ディレクトリ構造を維持したままコピー（例: `node_modules` 内の設定ファイルなど）
- Cobra フレームワークによる柔軟なフラグ指定
- Go 言語による単一バイナリでの動作
- ブランチ名に基づくパスの自動生成（リポジトリルートと同じ階層に `{プロジェクト名}-{ブランチ名}`）。カスタムパスを指定するには `--path`、ベースとなるブランチを指定するには `--base`、強制的にデフォルトブランチをベースにするには `--from-default` を使用可能。
- 一覧表示（`list` / `ls`）および削除（`remove` / `rm`）機能のサポート
- **ポート管理**: 各ワークツリーに一意のポートインデックスを自動割り当てし、衝突を防止
- **カスタムフック**: ワークツリーの作成（`add`）や削除（`rm`）に連動して、複数のシェルコマンドを実行
- **設定の可視化**: `config` サブコマンドで最終的なマージされた設定を確認したり、構文チェックを行ったりできます。
- **双方向の同期**: ワークツリーで編集した無視ファイルをルートプロジェクトに同期できる `sync` サブコマンド
- **Agent Skill**: AI エージェント向けの専門スキルを配布・インストール可能

## Badges

[![Release](https://img.shields.io/github/release/mocyuto/zgt.svg?style=for-the-badge)](https://github.com/mocyuto/zgt/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Build status](https://img.shields.io/github/actions/workflow/status/mocyuto/zgt/ci.yml?style=for-the-badge&branch=main)](https://github.com/mocyuto/zgt/actions?workflow=ci)

## インストール

### Homebrew

macOS での最も簡単なインストール方法は Homebrew を使用することです:

```bash
brew install mocyuto/tap/zgt
```

### ビルド

```bash
go build -o zgt .
```

### パスの設定

バイナリを `PATH` の通った場所に配置して使用します。

```bash
# mac / linux の例
sudo mv zgt /usr/local/bin/zgt
```

## 開発フローと挙動

`zgt` は単なるコマンドのラッパーではなく、開発者のコンテキスト切り替えをシームレスにすることを目的としています。

### 1. 新しい機能の開発を始める (`add`)

`git worktree add` を実行すると、通常は `.env` などの設定ファイルが欠落した状態で新しいディレクトリが作成されます。`zgt` は以下の手順を自動化します：

1. **ワークツリーの作成**: 指定されたパスにディレクトリを作成し、ブランチをチェックアウトします。
2. **自動パス生成**: ブランチ名だけを指定すれば、リポジトリルートと同じ階層に `{プロジェクト名}-{ブランチ名}` として自動で配置されます。`--path` または `-p` でカスタムパスを指定することも可能です。
3. **設定の同期**: 元のディレクトリにある「無視されているファイル（`.env` など）」を特定し、構造を維持したままコピーします。
4. **ポートの割り当て**: そのワークツリー専用の「ポートインデックス」を予約します。
5. **自動セットアップ**: フック（`hooks.add`）に `npm install` などを設定しておけば、依存関係の解決まで自動で行われます。

```bash
# feature-login ブランチを新しいワークツリーで開始（自動パス）
zgt add feature-login

# 特定のブランチをベースにして開始
zgt add feature-login --base develop

# 強制的にデフォルトブランチをベースにして開始
zgt add feature-login --from-default

# カスタムパスを指定して開始
zgt add feature-login --path ./experimental-worktree
```

### 2. 複数プロジェクトを同時に動かす (`env` / `ports`)

複数のワークツリーでサーバーを同時に立ち上げる際、ポートの衝突が問題になります。`zgt` はこれを解決します。

- **挙動**: 各ワークツリーには `0, 1, 2...` とインデックスが振られます。
- **動的な計算**: ポート番号は、そのワークツリーのプロジェクトルートにある `zgt.config.yml` の `ports` 設定（ベースポート）にインデックスを加算して算出されます。これにより、プロジェクトごとに異なるベースポートを設定していても、正しく計算されます。
- **活用**: 設定ファイルで `api: 8080` と定義しておけば、`zgt env` はそのワークツリーに応じたポート（`8080`, `8081`...）を環境変数として出力します。

```bash
# ワークツリーに移動して環境変数を読み込む
cd ../my-project-feature-login
eval "$(zgt env)"

# これで $API_PORT が 8081 など、他のワークツリーと被らない値になります
npm start
```

### 3. 作業が終わったら片付ける (`remove`)

作業が完了したワークツリーの削除も一括で行えます。

- **挙動**: ワークツリーのディレクトリを削除すると同時に、関連するローカルブランチも削除します（オプションで変更可能）。
- **クリーンアップ**: ポートインデックスも返却され、再利用可能になります。削除後にフック（`hooks.rm`）を実行することも可能です。
- **ブランチを切り替えた場合**: ワークツリー内で別ブランチに切り替えて作成時のブランチとズレた場合、`zgt rm <元のブランチ名>` を実行するとフォルダも削除するか確認します。元のブランチ名は `zgt add` 時に `zgt state` に記録され（旧バージョンで作成したワークツリーは自動パスから推測）、シェル補完にも表示されます。`y` でワークツリーフォルダごと削除し、それ以外は元のブランチのみ削除します。フォルダでチェックアウト中のブランチは削除されません。

```bash
# 作業完了。ディレクトリとブランチを削除
zgt rm feature-login
```

### 4. 無視ファイルの同期 (`sync`)

ワークツリー内で `.env` などの設定ファイルを編集し、その内容をメインプロジェクトのルートディレクトリに反映させたい場合：

- **対話モード**: `zgt sync` を実行すると TUI（`rivo/tview` を利用）が起動し、同期したいファイルを個別に選択できます。
- **一括同期**: `-a` / `--all`（または `--force`）フラグを使用すると、すべての無視ファイルを即座に同期します。
- **パスフィルタリング**: `-p` / `--path` を使用して、ファイル一覧を特定のパスや文字列で絞り込むことができます。
- **除外フィルタリング**: `-i` / `--ignore` を使用して、指定したパスや文字列を含むファイルを除外できます（カンマ区切りで複数指定可能）。
- **サブディレクトリ実行対応**: サブディレクトリ内で `zgt sync` を実行した場合も、同期先では相対パスが維持されます。たとえば `src` 配下で `.env` を同期すると、メインプロジェクト側でも `src/.env` が更新されます。

```bash
# 変更内容を選択してルートに同期
zgt sync

# 特定のパスにマッチするファイルのみ同期
zgt sync --path internal/config

# 特定のパスを除外して同期
zgt sync --ignore "node_modules,temp"
```

### 5. 割り当て状況を確認する (`list` / `ports`)

- `list`: どのワークツリーがどのブランチで動いているか、GitHub PR の状態や未コミットの変更があるか（`[DIRTY]`）を表示します。割り当てられているポート番号も一覧できます。
- `ports`: 現在どのパスにどのインデックス・ポート番号が割り当てられているかを確認できます。`-a` / `--all` フラグを使うと、現在のプロジェクトだけでなく、すべてのプロジェクトの割り当て状況を表示します。

### 6. その他のコマンド

- `tmux ls`: tmux のセッション、ウィンドウ、ペインの状態（Running/Waiting）をツリー形式で表示します。ペイン内で opencode または Claude Code のセッションが動いている場合は `[agent: status]` バッジも表示します（[AIエージェントステータス](#7-aiエージェントステータス-opencode--claude-code) 参照）。
- `tmux open`: 指定したワークツリーの tmux ウィンドウを開く、またはアクティベートします。ウィンドウが存在すれば切り替え、存在しなければ設定に従って作成します。引数を指定しない場合、インタラクティブな TUI が表示され、1つ以上のワークツリーを選択して開くことができます。`--profile <名前>` でウィンドウ名やペインコマンドに使用するプロファイルを上書き（永続化）できます。`zgt add --profile` と同様の挙動で、TUI モードでは選択したすべてのワークツリーに同じプロファイルが適用されます。
- `tmux close`: 指定したワークツリーの tmux ウィンドウを正常に閉じます（SIGTERM を送信して終了を待ちます）。
- `ports update`: 現在のプロジェクトのポート割当を最新の設定に基づいて更新します。不足している割当の追加や、設定から消えた割当の削除を自動で行います。
- `config`: マージされた最終的な設定（グローバル + プロジェクト + フラグ）をYAML形式で表示します。`--check` フラグを使うことで設定ファイルの構文エラーをチェックでき、`--raw` フラグを使うことでプレースホルダー置換をスキップできます。
- `config edit`: システムエディタを使用して設定ファイルを編集します。デフォルトでローカルプロジェクトの設定を編集します。
- `version`: `zgt` のバージョン番号を表示します。
- `skill install`: カレントリポジトリの `skills/` ディレクトリ内のスキルを、グローバルなエージェントスキルディレクトリ (`~/.claude/skills/`) にインストールします。
- `agent status`: 各ワークツリー内の opencode / Claude Code セッションが `working`（作業中）、`idle`（入力待ち）、`ask`（権限/回答待ち）のいずれかを表示します。`-w` / `--watch` で2秒ごとに更新、`-a` / `--all` でセッションのないワークツリーも含めて表示します。
- `agent install [claude|opencode|all]`: セッションステータスを `zgt` に報告する Claude Code フックと opencode プラグインをインストールします。マシンごとに1回実行します。再実行も安全です。
- `agent uninstall [claude|opencode|all]`: `agent install` でインストールしたフック/プラグインを削除します。

### 7. AIエージェントステータス (opencode / Claude Code)

`zgt` は、ワークツリー内で動いている AI コーディングエージェントが「作業中」「入力待ち（idle）」「権限/回答待ち（ask）」のいずれかを一目で分かるようにします。次の2箇所に表示されます。

- `zgt tmux ls` は各ペインに色付きの `[agent: status]` バッジを追加します（緑=working、シアン=ask、灰=idle）。
- `zgt agent status` は全ワークツリーの現在のステータス（色付き）と経過時間を一覧表示します。

ステータスは `zgt agent install` がセットアップするフック/プラグインから書き込まれます。

- **Claude Code**: `UserPromptSubmit` → `working`、`Stop` → `idle`、`Notification` → `ask`、`SessionEnd` → クリア、の各コマンドフック。フックは `zgt agent hook claude` を呼び出し、stdin のイベント JSON を読み取ります。
- **opencode**: プラグイン（`~/.config/opencode/plugins/zgt-status.js`）がセッションライフサイクルと permission イベントを `working`/`idle`/`ask` に対応付け、`session.deleted` でレコードをクリアします。`permission.asked` / `permission.updated` はバージョンによりどちらかが発火するため両方を受け付けます。`session.idle` 到着から500ms以内の遅延 `message.updated` および遅延 `permission.replied` イベントは無視され、ステータスは `idle` に維持されます。遅延 `message.part.updated`（question 完了/エラー）はセッションが既に idle のときは時間経過に関わらず無視されます（プラン/ワークフロー回答はデバウンス期間を超えて到着することが多いため）。次のターンは主に `session.status:busy`/`retry` で検出し、デバウンス期間後の user `message.updated` をフォールバックとします。

```bash
# 初回セットアップ: 両エージェントのフック + プラグインをインストール
zgt agent install

# 全ワークツリーのステータスを確認
zgt agent status

# リアルタイム更新ビュー
zgt agent status --watch

# いずれか一方だけインストール
zgt agent install claude
zgt agent install opencode

# 不要になったら削除
zgt agent uninstall
```

ステータスレコードは `~/.config/zgt/agent-status/` にワークツリーのパスをキーとして保存され、設定した `stale_after` 期間（デフォルト `1h`）を過ぎると自動的に破棄されるため、クラッシュ/強制終了したセッションが永遠に「working」のままになることはありません。

## 設定

`zgt` は以下の優先順位で設定ファイルを読み込みます（YAML形式）。

1. プロジェクトごとの設定（プロジェクトルートの `zgt.config.yaml` または `.yml`）
2. グローバル設定（`~/.config/zgt/config.yaml`）
3. `--config` フラグで明示的に指定されたパス

### 設定ファイルの生成 (`init`)

プロジェクトのルートディレクトリで以下のコマンドを実行することで、デフォルトの設定ファイル（`zgt.config.yml`）を生成できます。

```bash
zgt init
```

このコマンドは以下の動作を行います：

1.  ポート管理や環境変数のテンプレート、サンプルフックを含む設定ファイル（`zgt.config.yml`）を作成します。
2.  `zgt.config.yml` および `zgt.config.yaml` を自動的に `.gitignore` ファイルに追記し、誤ってコミットされないようにします。

すでに `zgt.config.yml` または `zgt.config.yaml` が存在する場合は、上書きを防止するためにスキップされます。

### 設定の編集 (`config edit`)

CLI から直接、好みのエディタ（`$EDITOR` または `vi`）を使用して設定ファイルを編集できます。

```bash
# ローカル設定を編集
zgt config edit --local

# グローバル設定を編集
zgt config edit --global
```

保存前に YAML の構文チェックが自動的に行われます。

### 設定 (Configuration)

`zgt` はグローバル (`~/.config/zgt/config.yaml`) またはローカル (プロジェクトのルートにある `zgt.config.yml`) で設定可能です。

```yaml
# 利用可能な設定オプションの例
add:
  from_default: true # 新しい worktree を常にデフォルトブランチ (例: main) から作成する
  auto_pull: true # worktree 作成前にデフォルトブランチの更新を pull する（未コミットの変更は自動的に stash されます）

ignore:
  - "*.tmp"
  - "local-debug.log"
  - ".env"

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
  keep_open: true # tmux ウィンドウを自動的に閉じないようにする (デフォルト: false)
  panes:
    - id: main
      commands: ["yarn"]
    - id: dev
      target: main
      split: horizontal
      size: 50%
      commands: ["yarn dev"]

agent:
  enabled: true       # `tmux ls` と `agent status` で opencode / Claude Code のステータスバッジを表示 (デフォルト: true)
  stale_after: "1h"   # この期間より古いステータスレコードは破棄 (デフォルト: "1h")
```

- `WEB_PORT=3001`

### カスタム環境変数

`env` セクションで独自の環境変数を定義できます。これらの値にはプレースホルダーを使用でき、`zgt env` を介して自動的にエクスポートされます。

```yaml
env:
  COMPOSE_PROJECT_NAME: "zgt-{{.Repo}}"
  DEBUG: "true"
```

実行例：

```bash
eval "$(zgt env)"
echo $COMPOSE_PROJECT_NAME # zgt-myrepo
```

これらの環境変数は、[カスタムフック](#カスタムフック) の実行時にも適用されます。

### カスタム無視パターン

コピー時に除外したいファイルパターンを `ignore` セクションで指定できます。

```yaml
ignore:
  - ".env.production"
  - "secrets/*"
```

### 削除後のデフォルトブランチPULL

ワークツリー削除後に、リモートからデフォルトブランチを自動的にプルするには、`rm` フックに git コマンドを追加します。

```yaml
hooks:
  rm:
    - "git pull origin main:main"
```

### Git hooks の自動登録

新しく作成した worktree ごとに、Git hooks ディレクトリを自動登録できます。

```yaml
git_hooks:
  enabled: true
  path: .githooks
  shared: true
```

- `path` には絶対パスも指定できます。なお、`zgt` は解決されたパスをワークツリーの Git 設定に**絶対パス**として登録します。これにより、どのディレクトリから Git コマンドを実行してもフックが正しく動作します。
- `shared` (boolean):
  - `true` (デフォルト): 相対パスはメインプロジェクトのルート基準で解決されます。すべての worktree で同じ hooks ディレクトリを共有します。
  - `false`: hooks ディレクトリが元のリポジトリから新しい worktree にコピーされます。各 worktree は独立したコピーされた hooks を使用します。
- worktree 側に既に `core.hooksPath` が設定されている場合、`zgt add` は上書きせず警告だけ出します。

### カスタムフック

フックを使用すると、ワークツリーの管理時にシェルコマンドを自動実行できます。

```yaml
hooks:
  # 'add'（作成）後に実行するコマンド
  add:
    - "tmux new-window -n [{{.Repo}}]{{.Branch}} -c {{.Path}}"
    - "echo 'Welcome to {{.Repo}}'"
  # 'remove'（削除）後に実行するコマンド
  rm:
    - "echo 'Cleanup for {{.Branch}}'"
```

#### 使用可能なプレースホルダー

プレースホルダーは `hooks`、`env`、および `tmux.window_name` で利用可能です。

| プレースホルダー     | 説明                                             |
| :------------------- | :----------------------------------------------- |
| `{{.Path}}`          | ワークツリーディレクトリの絶対パス               |
| `{{.Repo}}`          | メインプロジェクトのルートディレクトリ名         |
| `{{.CurrentDir}}`    | 現在の作業ディレクトリ名                         |
| `{{.Branch}}`        | 対象ブランチ名（引数で指定されたもの）。`zgt add` で作成したワークツリーでは、ワークツリー内でブランチを切り替えても作成時のブランチ名のまま |
| `{{.TargetBranch}}`  | `{{.Branch}}` のエイリアス                       |
| `{{.CurrentBranch}}` | `zgt` 実行時にチェックアウトされていたブランチ名 |
| `{{.Profile}}`       | 選択されたプロファイル名（デフォルト時は空）。[プロファイル](#プロファイルprofiles) 参照 |

### プロファイル（Profiles）

プロファイルを使うと `zgt add <branch> --profile <name>` でワークツリーごとに `env` の上書きと `hooks` の追加が行えます。選択されたプロファイルは `zgt` の state に保存されるので、`zgt env`・`zgt rm`・対応する hooks がその worktree のライフサイクル全体でプロファイルを使い続けます。

プロファイルはトップレベルの `profiles` マップの下に定義します。プロファイルの `env` はトップレベルの env を**キー単位**で上書きします（明示的に指定していないキーはトップレベルの値を引き継ぎます）。プロファイルの `hooks.add` / `hooks.rm` は**トップレベルの hooks の後に追加**実行されるので、デフォルトの挙動はそのままに追加ステップだけを差し込めます。

```yaml
env:
  COMPOSE_PROJECT_NAME: "{{.Repo}}-{{.Branch}}"
  DB_HOST: "db"

profiles:
  migration:
    env:
      # DB_HOST だけ上書き。COMPOSE_PROJECT_NAME はトップレベルを継承。
      DB_HOST: "iso-db"
    hooks:
      add:
        # トップレベルの `docker compose up -d api envoy` などが終わった後に実行。
        - "./scripts/db-clone.sh"
      rm:
        # トップレベルの `docker compose down` の後に追加。
        - "docker compose down -v"
```

使い方:

```bash
# 通常の worktree（共有 DB を参照）:
zgt add feat/foo

# 隔離 DB を持つ worktree（`add` で DB の volume コピー、`rm` で volume ごと削除）:
zgt add feat/migrate-bar --profile migration
```

`--profile` 未指定や `--profile default` は暗黙のデフォルトプロファイル（トップレベルのみ）を選択するので、後方互換性は完全に保たれます。未知のプロファイル名は `zgt add` 時にエラーになります。「メインworktree が DB を所有し、migration worktree は Docker volume をコピーして独立 DB を持つ」運用の完全なサンプルは `example/` ディレクトリ（`docker-compose.yaml` + `zgt.config.yaml` + `scripts/db-clone.sh`）を参照してください。

#### プロファイルでの `tmux` 上書き

プロファイルには `tmux` セクションも記述でき、トップレベルの `tmux` にフィールド単位で重ね合わせます。マージ規則は以下の通りです。

- `enabled` / `keep_open`: OR（論理和）。プロファイル側で `true` にすると有効化されますが、`false` で無効化することはできません。そのため、トップレベルで tmux 無効でもプロファイルで有効化できますし、`zgt rm` でウィンドウを残す設定もプロファイル側で強制できます。
- `window_name`: プロファイルが空でない値を設定した場合に上書きします。tmux ウィンドウリストでプロファイルを区別する（例: `[repo]fe-branch` と `[repo]branch`）ための主要なレバーです。
- `panes`: プロファイルが1つでもペインを定義した場合は全体を置き換え、未定義ならトップレベルの `panes` をそのまま使用します。

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

プロファイル解決後の `tmux` 設定は `zgt add`・`zgt tmux open`・`zgt tmux close` で共通して使われます（`tmux close` も `zgt rm` と同様に `zgt state` からプロファイルを解決するようになりました）。なお `zgt tmux close` は `keep_open` を無視して強制終了するため、プロファイルで `keep_open: true` にしても `zgt rm` では窓が残りますが `zgt tmux close` では閉じられます。

プロファイルがグローバル設定とローカル設定の**両方**に定義されている場合も同じフィールド単位の規則が適用されます。`enabled`/`keep_open` は両プロファイル定義の OR、`window_name` はローカルプロファイルが空でない値を設定すれば上書き、`panes` はどちらか側が1つでも定義すれば置換（ローカルがグローバルを置換、どちらも未定義なら上位の設定を継承）されます。

#### テンプレート関数

プレースホルダーの値を変換するための関数を使用できます。

| 関数       | 説明                                                    | 例                                     |
| :--------- | :------------------------------------------------------ | :------------------------------------- |
| `hostname` | ホスト名として無効な文字（`_`, `/`）を `-` に置換する。 | `{{.Branch \| hostname}}`->`feat-test` |

### Tmux 連携

`zgt` は、`add` を実行したときに自動的に tmux のウィンドウを作成し、複数のペインに分割してコマンドを実行することができます。`id` を指定することで、特定のペインをターゲットにして分割することが可能です。

```yaml
tmux:
  enabled: true
  panes:
    - id: main
      commands: ["yarn"]
    - id: side
      target: main # main を左右に分割
      split: horizontal
      size: 50%
      commands: ["yarn dev"]
    - target: side # side を上下に分割
      split: vertical
      commands: ["yarn watch"]
    - target: main # main を上下に分割
      split: vertical
      commands: ["tail -f logs/app.log"]
```

#### ペインのプロパティ

| プロパティ    | 説明                                                                            |
| :------------ | :------------------------------------------------------------------------------ |
| `enabled`     | tmux 連携を有効にするかどうか (`true` / `false`)。                              |
| `keep_open`   | `zgt rm` 実行時にウィンドウを閉じないようにするかどうか (デフォルト: `false`)。 |
| `window_name` | (任意) tmux ウィンドウ名のプレースホルダー。                                    |
| `id`          | (任意) `target` から参照するための、ペインの一意識別子。                        |
| `target`      | (任意) 分割対象とするペインの ID。省略時は最後に作成されたペイン。              |
| `commands`    | そのペインで実行するコマンドのリスト。                                          |
| `split`       | 分割方向: `horizontal` (h) または `vertical` (v)。                              |
| `size`        | ペインのサイズ (例: `20%` でパーセント指定、`20` で行/列数指定)。               |

`enabled` が `true` の場合、`zgt` は以下の動作を行います：

1. `window_name` で指定された名前（デフォルトは `[repo]branch`）の新しい tmux ウィンドウを作成します。
2. `panes` のリストに従って分割を作成します。各分割は指定された `target` または直前に作成されたペインを対象とします。
3. 各ペインで `commands` を実行し、終了後もシェルを開いたままにします。

#### 注意事項

- `tmux` がインストールされており、tmux セッションが実行中である必要があります。

#### 補足

- 単一のコマンドのみ実行する場合は、リストではなく文字列で指定可能です： `add: "echo hello"`
- コマンドは `/bin/sh -c` を介して実行されるため、パイプやステータスチェックも利用可能です。

## Agent Skill の管理

`zgt` は、AI エージェント向けの「エキスパートスキル」を管理する機能を提供します。スキルとは、エージェントの能力を拡張するための指示書やリソースのセットです。

### スキルのインストール (`skill install`)

カレントリポジトリにあるスキルをインストールして、AI コラボレーター（Claude など）が利用できるようにします。

```bash
zgt skill install
```

実行すると、以下の 4 つのインストール先からインタラクティブに（チェックボックス形式で）選択できます。

- **Local .claude**: `./.claude/skills/`
- **Local .agents**: `./.agents/skills/`
- **Global .claude**: `~/.claude/skills/`
- **Global .agents**: `~/.agents/skills/`

`-a` または `--all` フラグを使用すると、すべてのターゲットに即座にインストールされます。

## 注意事項

- コピー対象は、実行時のワークツリールートにおいて `git ls-files --others --ignored --exclude-standard` でリストアップされるファイルです。
- 実行には `git` と　`gh` がインストールされている必要があります。

## 開発・テスト

ツールチェーン（Go・Bun）のバージョンは [`mise.toml`](./mise.toml) で管理しています。初回は `mise install` を実行し、以降は mise タスクを使います。

```bash
mise install
mise run test       # Go テスト: go test ./...
mise run test-js    # JS テスト: bun test internal/agentstatus/assets/
mise run test-all  # 両方を並列実行
```

直接コマンドを実行することも可能です:

```bash
go test -v ./...
bun test internal/agentstatus/assets/
```

opencode 用ステータスプラグイン（`internal/agentstatus/assets/zgt-status.js`）の状態遷移テストは同ディレクトリにあり、プラグインを編集したら `mise run test-js` で確認してください。

## よくある質問 (FAQ)

### Q: すでに存在するブランチを `add` した場合どうなりますか？

`zgt` はローカルまたはリモート（`origin`）に同名のブランチがあるか確認します。

- 存在する場合、その既存のブランチを使用して新しいワークツリーを作成します。
- ただし、Git の仕様により、**同じブランチを複数のワークツリーで同時にチェックアウトすることはできません**。すでに他で使われている場合は Git のエラーが表示されます。

### Q: `rm` コマンドを実行すると、作成されたディレクトリは消えますか？

はい、`zgt rm`（または `remove`）を実行すると以下の処理が行われます：

1. ファイルシステム上のワークツリーディレクトリを削除します。
2. Git の管理からワークツリーを登録解除します。
3. 関連付けられたローカルブランチを削除します（`--keep-branch` を指定しない場合）。
   未コミットの変更がある場合は、Git が保護機能として削除をブロックします。その場合は `-f`（`--force`）オプションで強制的に削除できます。
