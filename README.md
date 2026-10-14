# AI Vault

[日本語](README.ja.md) | [English](README.md) | [한국어](README.ko.md) | [简体中文](README.zh-CN.md)

**Lightweight, local-first AI memory and a Markdown knowledge base for Codex, Claude, and other AI clients.** Headless note-taking through an MCP server and JSON CLI, shipped as one executable for Windows, Linux, and macOS.

Obsidian-style `[[links]]`, backlinks, tags, search, and note history without an editor, a database, or a background web server. Your notes stay readable as ordinary UTF-8 Markdown. No API key, account, Git, Node.js, Python, or Go installation is needed to run a release binary.

Notes support UTF-8 Japanese, English, Korean, and Chinese, including mixed-language content. AI Vault preserves note text without automatic translation. CLI help, command flags, JSON fields, and configuration keys stay in English. These guides are available in four languages; the main README and internal metadata use English.

## Give this URL to your AI

Repository: **<https://github.com/uyuk216/ai-vault>**

Paste this into a local coding AI that can run terminal commands:

```text
Install AI Vault from https://github.com/uyuk216/ai-vault on this computer.
Read its README and installer first. Download the release for this OS/CPU,
verify SHA-256, and use ~/AI-Vault as the shared vault. Connect the installed
Codex and Claude Code clients with the installer’s auto-connect option.
For my other local MCP clients, use the generated absolute-path configuration
without replacing unrelated settings. Verify the CLI and MCP tools can list
notes, create a small test note, read it, and move that test note to trash.
Explain any client that still needs a restart or manual setup.
```

A repository URL gives your AI the instructions and downloads. It is **not a universal MCP install URL**: the AI needs local terminal access and permission to configure the client. Chat-only or hosted clients cannot start this local executable just by receiving a link.

## Install

Prebuilt releases support **Windows / Linux / macOS × x86-64 / ARM64**. Linux needs a standard Linux environment; these are native binaries, not browser or mobile applications.

### macOS / Linux

Requires `curl`, `tar`, and one SHA-256 tool (`sha256sum`, `shasum`, or `openssl`). These are normally already available on macOS and common Linux installations.

```sh
curl -fsSL https://raw.githubusercontent.com/uyuk216/ai-vault/v0.1.0/install.sh | sh -s -- --version v0.1.0 --auto-connect
```

Installs to `~/.local/bin/aivault`; notes go to `~/AI-Vault`. No administrator access is needed. Use `~/.local/bin/aivault` directly, or add `~/.local/bin` to your shell's `PATH`.

### Windows PowerShell

Requires Windows PowerShell 5.1 or PowerShell 7. Run as your ordinary user:

```powershell
& ([scriptblock]::Create((Invoke-WebRequest -UseBasicParsing 'https://raw.githubusercontent.com/uyuk216/ai-vault/v0.1.0/install.ps1').Content)) -Version v0.1.0 -AutoConnect
```

Installs to `%LOCALAPPDATA%\AI-Vault\bin\aivault.exe`; notes go to `%USERPROFILE%\AI-Vault`. Use the printed executable path, or add its directory to your user `PATH`.

The commands above run the tagged installer. To inspect it first, download `install.sh` or `install.ps1` from that tag and run the saved file. Both installers download the binary and `SHA256SUMS` from **the same release tag**, and stop on a checksum mismatch. Checksums detect a changed download; they do not independently prove who published it.

### Installation options

```sh
sh install.sh --version v0.1.0 --install-dir "$HOME/.local/bin" --vault "$HOME/AI-Vault" --auto-connect
```

```powershell
.\install.ps1 -Version v0.1.0 -InstallDir "$env:LOCALAPPDATA\AI-Vault\bin" -Vault "$env:USERPROFILE\AI-Vault" -AutoConnect
```

`AIVAULT_VERSION`, `AIVAULT_INSTALL_DIR`, and `AIVAULT_HOME` provide the same defaults. Without a version, a saved installer resolves the latest GitHub Release once, then uses that tag for both downloads. Omit `--auto-connect` / `-AutoConnect` to install and print configuration without registering clients.

Before updating, stop clients that are running AI Vault. Re-run the installer for the version you want; existing notes are retained. To uninstall, remove the executable and its MCP entries. Keep or back up your vault folder separately.

## Connect your AI clients

`--auto-connect` / `-AutoConnect` detects **Codex CLI** and **Claude Code CLI** in `PATH` and uses their supported `mcp add` commands. Claude Code receives a user-scoped entry, available across its projects. The installer preserves an existing `ai-vault` entry and tells you to inspect it. It never replaces an entire client configuration file. Existing entries may use a different vault; check them before assuming all clients share notes.

Restart or reconnect your client after registration. Its own trust and tool-approval settings still apply.

Manual setup, using the actual absolute paths printed by the installer:

```sh
codex mcp add ai-vault -- /absolute/path/aivault --vault /absolute/path/AI-Vault serve
claude mcp add --scope user --transport stdio ai-vault -- /absolute/path/aivault --vault /absolute/path/AI-Vault serve
```

On Windows, use the `.exe` path and quote paths containing spaces. For example, in PowerShell:

```powershell
codex mcp add ai-vault -- "$env:LOCALAPPDATA\AI-Vault\bin\aivault.exe" --vault "$env:USERPROFILE\AI-Vault" serve
claude mcp add --scope user --transport stdio ai-vault -- "$env:LOCALAPPDATA\AI-Vault\bin\aivault.exe" --vault "$env:USERPROFILE\AI-Vault" serve
```

For Claude Desktop and other **local stdio MCP clients**, generate the exact JSON:

```sh
aivault --vault /absolute/path/AI-Vault config --format json
```

Merge the generated `ai-vault` server entry into that client's configuration through its documented settings. Do not replace its other entries. A typical configuration looks like:

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

For a Codex TOML snippet, use `aivault config --format codex`. Every client should use **the same absolute vault path**. There is no localhost port or cloud endpoint: the client starts and stops the stdio process as needed. Run `aivault serve --read-only` when a client only needs to read knowledge.

Connection references: [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli), [Claude Code MCP](https://code.claude.com/docs/en/mcp), [local MCP server setup](https://modelcontextprotocol.io/docs/develop/connect-local-servers).

### Verify the installation

```sh
aivault version
aivault doctor
aivault list
aivault write setup-check.md --content '# Setup check'
aivault read setup-check.md
aivault trash setup-check.md --expected-sha HASH_FROM_READ
```

Use the `sha256` returned by `read` for `HASH_FROM_READ`. Choose a different test filename if this one already exists. Then ask your restarted AI client to call `vault_list`, create a uniquely named test note with `vault_write`, read it with `vault_read`, and trash it with `vault_trash` using its current hash. A working CLI alone does not prove the client has connected to MCP.

## Add knowledge from a GitHub URL

This is separate from installing the tool:

```sh
aivault import https://github.com/OWNER/REPOSITORY
aivault import https://github.com/OWNER/REPOSITORY --prefix sources/reference
```

The importer downloads a public GitHub repository at a captured commit and copies Markdown notes into `sources/OWNER/REPOSITORY/` by default. It never executes repository code. The JSON report includes the source, commit, imported paths, and skipped paths. Identical existing notes are skipped; conflicting existing notes cause an error rather than an overwrite. Private repositories and arbitrary download URLs are not supported.

Ask your connected AI: “Import the Markdown knowledge from `https://github.com/OWNER/REPOSITORY` into AI Vault, then summarize the imported notes.”

Treat imported notes as outside content. They may contain inaccurate information or instructions aimed at the AI. Note text is evidence to read, not permission to execute commands or change policy.

## Daily use

The commands below assume `aivault` is in `PATH`. Otherwise substitute the absolute executable path from the installer. JSON goes to stdout; errors go to stderr. Put the global `--vault PATH` option **before** the command, or set `AIVAULT_HOME`.

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

Markdown can contain `[[another-note]]`, `[[folder/note|label]]`, Markdown links, and `#tags`. There is no embedding model: search is a case-insensitive literal text search. Ordinary Markdown files remain usable with text editors and Obsidian; Obsidian plugins and every Markdown extension are outside this tool's scope.

### Categories grow with your knowledge

Categories are ordinary relative folders. AI Vault does not pre-register subjects, programming languages, or a fixed taxonomy. The catalog reflects the folders that exist in your vault:

```sh
aivault categories
aivault write decision.md --category work/decisions --content '# A decision and its reasons'
aivault read work/decisions/decision.md
```

`work/decisions` is only an example. Supply any portable category path that fits your knowledge; the folder is created when the first note is saved there. For connected AI clients, ask the AI to call `vault_categories`, choose the most specific relevant existing folder, and pass that folder as the optional `category` on `vault_write`. It can create a new category when the existing catalog does not fit. An explicit note path also works without a category option.

The connected AI chooses the category from the note's meaning. AI Vault stores that choice as a folder; it does not run a local classification model or add an inference runtime.

### Safe edits and recovery

`read` returns the note's `sha256`. Supply that hash when editing, appending, or trashing an existing note:

```sh
aivault write inbox/idea.md --content '# Updated idea' --expected-sha HASH_FROM_READ
aivault append inbox/idea.md --content 'An additional thought.' --expected-sha HASH_FROM_LATEST_READ
aivault history inbox/idea.md
aivault trash inbox/idea.md --expected-sha HASH_FROM_LATEST_READ
aivault restore REVISION_ID recovered/idea.md
```

Replace the uppercase placeholders with returned values. Creating a note requires no hash and refuses an existing path. A changed hash means another writer edited the note; read it again and reconcile the content. History keeps prior contents, trash is recoverable, and restore refuses to blindly replace an existing destination. `write` and `append` also accept `--file PATH` or piped stdin instead of `--content`. Append adds exactly the supplied text; include a newline when needed.

Mutations use an interprocess vault lock. A concurrent writer gets a readable error and should retry after re-reading. Vault-relative `.md` paths only: traversal, hidden paths, symlinks, and Windows reserved names are rejected. Each note is limited to 4 MiB. Back up the whole vault, including `.aivault`, if you want to retain history.

If `doctor` reports a write lock after a crash, first stop all AI Vault processes and confirm no writer is active. Only then remove the stale `.aivault/write.lock` directory and run `doctor` again. Do not remove a live writer's lock.

### MCP tools

| Tool | Purpose |
| --- | --- |
| `vault_list`, `vault_read`, `vault_search` | Find and read notes |
| `vault_categories` | Discover existing category folders |
| `vault_links`, `vault_backlinks` | Explore connections |
| `vault_write`, `vault_append` | Save notes with hash checks |
| `vault_history`, `vault_trash`, `vault_restore` | Review and recover revisions |
| `vault_import` | Import Markdown from a public GitHub repository |

Example request: “Search AI Vault for my deployment decisions. Read the relevant notes before answering, and save any new decision only when I ask.” AI Vault exposes storage tools; your client's instructions determine when to use them. Installing it does not automatically capture every chat.

## Shared folder, separate computers

AI clients on one computer share knowledge by opening the same vault folder. AI Vault does **not** include cross-device synchronization. You can manage backups or synchronization yourself, but a directory lock is not a distributed lock across independent synced copies. Avoid simultaneous writes on separate computers and resolve sync conflicts before continuing.

Notes are local plaintext, not encrypted. Connecting a hosted AI service may send selected note contents to that provider under its own policies. Do not put secrets into a vault you intend to share publicly.

## Build and verify

Developers need the Go version declared in `go.mod`:

```sh
go test ./...
go build -trimpath -ldflags='-s -w' -o aivault ./cmd/aivault
```

Release assets are named `aivault_VERSION_OS_ARCH.tar.gz` (`.zip` on Windows), with OS `darwin`, `linux`, or `windows` and architecture `amd64` or `arm64`. Archive roots contain `aivault` / `aivault.exe`. `SHA256SUMS` covers the archives. See [GitHub Releases](https://github.com/uyuk216/ai-vault/releases).

For offline installer tests, `AIVAULT_TEST_RELEASE_DIR` may point to an **absolute local directory** containing those release assets and `SHA256SUMS`; an explicit version is required. It does not accept a mirror URL. Use a temporary install/vault directory, and stub `codex` and `claude` through `PATH` when testing auto-connect so real client settings are untouched.

## License

[MIT](LICENSE). Dependencies retain their original licenses; see [third-party notices](THIRD_PARTY_NOTICES.md). AI Vault is independent of Obsidian, OpenAI, and Anthropic.
