# Architecture

AI Vault is a headless, local-first AI memory and Markdown knowledge base. Every client uses the same absolute vault folder. Notes remain ordinary UTF-8 files.

- `internal/vault`: confined file access, portable paths, dynamic category folders, literal search, wiki/Markdown links, tags, conflict hashes, interprocess mutation lock, saved revisions and recoverable trash.
- `internal/mcpserver`: official Go MCP SDK stdio server. CLI and MCP share the same storage operations. Read-only mode exposes discovery and reading tools only.
- `internal/importer`: bounded public GitHub archive downloads at an immutable commit. Markdown-only import, preflight conflicts, no execution.
- `cmd/aivault`: JSON CLI, absolute-path MCP configuration and diagnostics.

There is no GUI, database, embedding service, API key requirement, network listener, or always-running daemon. Categories are directories, created when notes are first saved. No subject or programming-language taxonomy is installed.

Updates require a SHA-256 hash from the previous read. Cooperating local writers use a directory lock. Updates and trash save the previous UTF-8 content to checksummed revision files; note replacements are staged in the same filesystem. This is a local filesystem design, not a distributed transaction or cloud sync protocol. External editors do not participate in its write lock.

GitHub imports preflight the archive before creating notes. A later filesystem error or independent writer can interrupt a batch; the returned report identifies completed writes. Existing notes are never blindly overwritten. Repository-root HTTPS URLs and public repositories are supported.

The source contains no user vault. Tagged releases provide native executables for Windows, Linux and macOS, each in x86-64 and ARM64 builds, with checksums and dependency notices. Installers inspect the OS and use supported client commands where available; a repository URL is an instruction source for a local coding AI rather than a universal MCP client configuration format.
