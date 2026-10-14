# AI Vault

[日本語](README.ja.md) | [English](README.md) | [한국어](README.ko.md) | [简体中文](README.zh-CN.md)

**Codex、ClaudeなどのAIで共有する、軽量なローカルAIメモリーとMarkdown知識ベースです。** 画面を持たないMCPサーバーとJSON CLIを、Windows・Linux・macOS向けの実行ファイル1つで提供します。

Obsidian風の `[[リンク]]`、バックリンク、タグ、検索、履歴に対応します。エディター、データベース、常駐Webサーバーは不要です。ノートは普通のUTF-8 Markdownなので、専用アプリがなくても読めます。リリース版の実行にはAPIキー、アカウント、Git、Node.js、Python、Goのインストールは必要ありません。

日本語・英語・韓国語・中国語のノートと、それらが混在した文章を保存できます。ノート本文を自動翻訳せず、そのまま保持します。ガイドは4言語で用意し、メインのREADMEと内部メタデータ、CLIのヘルプ、コマンドのフラグ、JSONのフィールド名、設定キーは英語に統一します。

## AIにこのURLを渡す

リポジトリ: **<https://github.com/uyuk216/ai-vault>**

PC上でターミナルを操作できるAIに、次の依頼文を貼り付けてください。

```text
https://github.com/uyuk216/ai-vault をこのPCにインストールして。
READMEとインストーラーを先に読み、OSとCPUに合うリリースをSHA-256検証して
入れて。同じPCのAIで共有する保存先は ~/AI-Vault にして。
インストール済みのCodexとClaude Codeにはauto-connectで接続して。
それ以外のローカルMCPクライアントには出力された絶対パスの設定を使い、
他の設定は維持して。CLIとMCPで一覧・小さなテストノートの作成・読取・
ゴミ箱への移動を確認して。再起動や手動設定が残るクライアントは明記して。
```

リポジトリのURLから、AIが導入手順と配布ファイルを確認できます。ただし、どのMCPクライアントにもURLだけで自動登録できる仕組みではありません。AIには、そのPCのターミナル操作とクライアント設定の権限が必要です。チャット専用・クラウド専用のAIは、リンクを受け取るだけではローカルの実行ファイルを起動できません。

## インストール

配布済み実行ファイルは **Windows / Linux / macOS × x86-64 / ARM64** に対応します。Linuxでは通常のLinux実行環境が必要です。ブラウザーやスマートフォン向けのアプリではありません。

### macOS / Linux

`curl`、`tar`、SHA-256ツールのいずれか（`sha256sum`、`shasum`、`openssl`）が必要です。macOSと一般的なLinux環境には通常含まれています。

```sh
curl -fsSL https://raw.githubusercontent.com/uyuk216/ai-vault/v0.1.1/install.sh | sh -s -- --version v0.1.1 --auto-connect
```

実行ファイルは `~/.local/bin/aivault`、ノートは `~/AI-Vault` に保存します。管理者権限は不要です。`~/.local/bin/aivault` を直接実行するか、`~/.local/bin` をシェルの `PATH` に追加してください。

### Windows PowerShell

Windows PowerShell 5.1またはPowerShell 7で、通常のユーザーとして実行します。

```powershell
& ([scriptblock]::Create((Invoke-WebRequest -UseBasicParsing 'https://raw.githubusercontent.com/uyuk216/ai-vault/v0.1.1/install.ps1').Content)) -Version v0.1.1 -AutoConnect
```

実行ファイルは `%LOCALAPPDATA%\AI-Vault\bin\aivault.exe`、ノートは `%USERPROFILE%\AI-Vault` に保存します。表示された実行ファイルのパスを使うか、そのディレクトリをユーザーの `PATH` に追加してください。

上記コマンドはタグで指定したインストーラーを実行します。先に内容を確認する場合は、そのタグの `install.sh` または `install.ps1` を保存してから実行してください。どちらも実行ファイルと `SHA256SUMS` を**同じリリースタグ**から取得し、チェックサムが一致しなければ停止します。チェックサムはダウンロードした内容の変化を検出しますが、それだけで公開者の身元を証明するものではありません。

### 保存先とバージョンを指定する

```sh
sh install.sh --version v0.1.1 --install-dir "$HOME/.local/bin" --vault "$HOME/AI-Vault" --auto-connect
```

```powershell
.\install.ps1 -Version v0.1.1 -InstallDir "$env:LOCALAPPDATA\AI-Vault\bin" -Vault "$env:USERPROFILE\AI-Vault" -AutoConnect
```

環境変数 `AIVAULT_VERSION`、`AIVAULT_INSTALL_DIR`、`AIVAULT_HOME` でも指定できます。保存したインストーラーをバージョン指定なしで実行すると、最新のGitHub Releaseを一度確認し、そのタグで実行ファイルとチェックサムを取得します。`--auto-connect` / `-AutoConnect` を省略すると、クライアントに登録せず、導入と設定の表示だけを行います。

更新前には、AI Vaultを実行しているクライアントを停止してください。希望するバージョンでインストーラーを再実行しても、既存ノートは保持します。削除する場合は実行ファイルとMCP登録を取り除き、ノートのフォルダは必要に応じて保存・バックアップしてください。

## AIクライアントに接続する

`--auto-connect` / `-AutoConnect` は `PATH` にある **Codex CLI** と **Claude Code CLI** を検出し、それぞれが提供する `mcp add` コマンドで登録します。Claude Codeにはユーザー単位で登録するので、複数のプロジェクトで利用できます。既存の `ai-vault` 登録は維持し、確認方法を表示します。他のサーバーを含む設定ファイル全体を置き換えません。既存の登録が別の保存先を参照している場合は、同じノートを共有する前に確認してください。

登録後はクライアントを再起動するか、MCPを再接続してください。クライアント自身の信頼・ツール承認設定はそのまま適用されます。

手動登録では、インストーラーが表示した実際の絶対パスを指定します。

```sh
codex mcp add ai-vault -- /absolute/path/aivault --vault /absolute/path/AI-Vault serve
claude mcp add --scope user --transport stdio ai-vault -- /absolute/path/aivault --vault /absolute/path/AI-Vault serve
```

Windowsでは `.exe` を指定し、空白を含むパスを引用符で囲みます。PowerShellの例:

```powershell
codex mcp add ai-vault -- "$env:LOCALAPPDATA\AI-Vault\bin\aivault.exe" --vault "$env:USERPROFILE\AI-Vault" serve
claude mcp add --scope user --transport stdio ai-vault -- "$env:LOCALAPPDATA\AI-Vault\bin\aivault.exe" --vault "$env:USERPROFILE\AI-Vault" serve
```

Claude Desktopなど、**ローカルのstdio MCP**に対応するクライアント向けのJSONは次のコマンドで生成します。

```sh
aivault --vault /absolute/path/AI-Vault config --format json
```

クライアントの正式な設定方法に従って、生成された `ai-vault` エントリーを追加してください。既存の他のエントリーは維持します。設定の例:

```json
{
  "mcpServers": {
    "ai-vault": {
      "command": "/absolute/path/aivault",
      "args": ["--vault", "/absolute/path/AI-Vault", "serve"]
    }
  }
}
```

Codex向けTOMLは `aivault config --format codex` で生成できます。ノートを共有するすべてのクライアントで、**同じ保存先の絶対パス**を指定してください。localhostのポートやクラウド接続先はなく、必要なときにクライアントがstdioプロセスを起動・終了します。読取専用なら `aivault serve --read-only` を使います。

公式の接続手順: [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli)、[Claude Code MCP](https://code.claude.com/docs/en/mcp)、[ローカルMCPサーバーの設定](https://modelcontextprotocol.io/docs/develop/connect-local-servers)。

### 導入を確認する

```sh
aivault version
aivault doctor
aivault list
aivault write setup-check.md --content '# Setup check'
aivault read setup-check.md
aivault trash setup-check.md --expected-sha HASH_FROM_READ
```

`HASH_FROM_READ` は `read` が返した `sha256` に置き換えます。同じテスト名がすでにある場合は別の名前を使ってください。その後、再接続したAIに `vault_list`、`vault_write`、`vault_read`、現在のハッシュを指定した `vault_trash` で、別名のテストノートを確認してもらいます。CLIが動くことと、AIクライアントのMCP接続が成功したことは別の確認です。

## GitHubのURLから知識を追加する

ツール自体のインストールとは別に、公開リポジトリのMarkdownを取り込めます。

```sh
aivault import https://github.com/OWNER/REPOSITORY
aivault import https://github.com/OWNER/REPOSITORY --prefix sources/reference
```

特定したコミットの公開GitHubリポジトリを取得し、既定では `sources/OWNER/REPOSITORY/` にMarkdownノートを保存します。リポジトリ内のコードを実行しません。JSONの結果には取得元、コミット、追加したパス、スキップしたパスが含まれます。同じ内容の既存ノートはスキップし、内容が異なる既存ノートは上書きせずエラーにします。非公開リポジトリや任意のダウンロードURLには対応しません。

接続済みのAIには「`https://github.com/OWNER/REPOSITORY` のMarkdownをAI Vaultに取り込み、追加された知識を要約して」と依頼できます。

取り込んだ文章は外部のデータとして扱ってください。誤情報やAIへの指示が含まれる場合があります。ノート本文は読む対象であり、コマンド実行やルール変更の許可にはなりません。

## 普段の使い方

以下は `aivault` が `PATH` にある場合の例です。ない場合は実行ファイルの絶対パスに置き換えてください。標準出力はJSON、標準エラー出力はエラー情報です。保存先の指定はコマンドより**前**に `--vault PATH` を置くか、`AIVAULT_HOME` を設定します。

```sh
aivault init
aivault doctor
aivault list
aivault write inbox/idea.md --content '# Idea: a shared AI memory'
aivault read inbox/idea.md
aivault search 'shared AI'
aivault tags
aivault categories
aivault links inbox/idea.md
aivault backlinks inbox/idea.md
```

`[[another-note]]`、`[[folder/note|label]]`、Markdownリンク、`#tags` が使えます。検索は大文字・小文字を区別しない文字列検索で、埋め込みモデルは使いません。ノートはテキストエディターやObsidianでも使えますが、ObsidianのプラグインやすべてのMarkdown拡張機能を再現するものではありません。

### 分類は必要に応じて増やす

分類は保存先からの相対フォルダです。教科、プログラミング言語などの分類を初期登録しません。既存のフォルダから分類一覧を取得します。

```sh
aivault categories
aivault write decision.md --category work/decisions --content '# A decision and its reasons'
aivault read work/decisions/decision.md
```

`work/decisions` は例にすぎません。知識に合う任意の分類パスを指定すると、最初のノートを保存するときにフォルダを作ります。AIには `vault_categories` で一覧を取得し、最も具体的で適切な既存分類を選んで、`vault_write` の任意項目 `category` に渡すよう依頼してください。合う分類がなければ、新しい分類を作れます。分類オプションを使わず、ノートの相対パスを直接指定することもできます。

文章の意味を判断するのは接続先のAIです。AI Vaultは、その判断をフォルダとして保存します。ローカルの分類モデルや追加の推論環境は不要です。

### 安全な編集と復元

`read` はノートの `sha256` を返します。既存ノートの編集、追記、ゴミ箱への移動には、そのハッシュを指定します。

```sh
aivault write inbox/idea.md --content '# Updated idea' --expected-sha HASH_FROM_READ
aivault append inbox/idea.md --content 'An additional thought.' --expected-sha HASH_FROM_LATEST_READ
aivault history inbox/idea.md
aivault trash inbox/idea.md --expected-sha HASH_FROM_LATEST_READ
aivault restore REVISION_ID recovered/idea.md
```

大文字の仮の値は実際の返却値に置き換えてください。新規作成はハッシュ不要で、既存パスへの保存を拒否します。ハッシュが変わっていれば、読み直して変更内容を調整してから保存します。変更前の内容は履歴に残り、ゴミ箱からも復元できます。復元先にノートがある場合、ハッシュなしの上書きは行いません。`write` / `append` は `--content` の代わりに `--file PATH` や標準入力も使えます。追記する文字列に改行が必要な場合は、本文に含めてください。

変更処理はAI Vaultのプロセス間ロックで調整します。同時に書き込もうとしたプロセスはエラーを受け取り、読み直してから再試行します。ノートは保存先からの相対 `.md` パスに限り、親への移動、隠しパス、シンボリックリンク、Windowsの予約名を拒否します。1ノートの上限は4 MiBです。履歴も保存する場合は `.aivault` を含むフォルダ全体をバックアップしてください。

クラッシュ後に `doctor` がロックを報告した場合は、すべてのAI Vaultプロセスを停止し、書込中のプロセスがないことを確認してから、残った `.aivault/write.lock` ディレクトリを削除してください。処理中のロックは削除しないでください。

### MCPツール

| ツール | 用途 |
| --- | --- |
| `vault_list`, `vault_read`, `vault_search` | ノートの検索と読取 |
| `vault_categories` | 既存分類フォルダの取得 |
| `vault_links`, `vault_backlinks` | ノート同士の関連を調べる |
| `vault_write`, `vault_append` | ハッシュを確認して保存 |
| `vault_history`, `vault_trash`, `vault_restore` | 履歴の確認と復元 |
| `vault_import` | 公開GitHubリポジトリのMarkdownを取り込む |

依頼例:「AI Vaultからデプロイに関する決定を検索し、関連ノートを読んでから答えて。新しい決定を保存するのは私が依頼したときだけにして」。いつ保存ツールを使うかは、クライアントへの指示で決まります。導入するだけですべてのチャットを自動記録するものではありません。

## 同じPCでの共有と、別のPCでの利用

同じPCのクライアントは、同じ保存先を開くことで知識を共有します。AI Vault自体には**PC間の同期機能はありません**。バックアップや同期を別途管理することはできますが、フォルダのロックは、別々のPCにある同期コピー同士を調整する分散ロックではありません。別PCからの同時書込を避け、同期の競合を解決してから続けてください。

ノートはローカルの平文ファイルで、暗号化されません。クラウドのAIにつなぐ場合、読み取られた文章がその提供元へ送信されることがあります。公開する予定の保存先には秘密情報を入れないでください。

## ビルドと検証

開発には `go.mod` に記載されたバージョンのGoを使います。

```sh
go test ./...
go build -trimpath -ldflags='-s -w' -o aivault ./cmd/aivault
```

配布ファイルは `aivault_VERSION_OS_ARCH.tar.gz`（Windowsは `.zip`）です。OSは `darwin` / `linux` / `windows`、CPUは `amd64` / `arm64` を指定します。アーカイブ直下に `aivault` / `aivault.exe` があり、`SHA256SUMS` がアーカイブを検証します。[GitHub Releases](https://github.com/uyuk216/ai-vault/releases)から取得できます。

オフラインのインストーラーテストでは、配布ファイルと `SHA256SUMS` を置いた**絶対パスのローカルディレクトリ**を `AIVAULT_TEST_RELEASE_DIR` に指定できます。明示的なバージョン指定が必要で、ミラーURLは使えません。導入先と保存先は一時ディレクトリにし、auto-connectのテストでは `PATH` 上の `codex` と `claude` をスタブに置き換えて、実際の設定への書込を避けてください。

## Windowsの実行制御

配布バイナリは未署名です。信頼された発行者の署名を必須とするWindowsの実行制御が有効な端末では、AI Vaultがブロックされる場合があります。その端末のポリシーが認める署名が必要です。インストーラーはWindowsのセキュリティ設定を変更しません。WindowsのCIでは、これらのバイナリを許可する環境で実行とインストールを検証しています。

## ライセンス

[MIT](LICENSE)。依存コードには元のライセンスが適用されます。[第三者のライセンス情報](THIRD_PARTY_NOTICES.md)も参照してください。AI VaultはObsidian、OpenAI、Anthropicとは独立したプロジェクトです。
