#!/bin/sh
# AI Vault release installer. No sudo, runtime, daemon, or client config rewriting.
set -eu

repo=uyuk216/ai-vault
version=${AIVAULT_VERSION:-}
install_dir=${AIVAULT_INSTALL_DIR:-"$HOME/.local/bin"}
vault_dir=${AIVAULT_HOME:-"$HOME/AI-Vault"}
auto_connect=0
tmp_dir=
staged_binary=

fail() { printf 'AI Vault: %s\n' "$*" >&2; exit 1; }
usage() {
    cat <<'USAGE'
Usage: sh install.sh [--auto-connect] [--version VERSION] [--install-dir DIR] [--vault DIR]

Downloads a release for this OS/CPU, verifies SHA-256, installs under your user,
and initializes a shared Markdown vault. --auto-connect registers ai-vault with
installed Codex and Claude Code CLIs. Existing ai-vault entries are preserved.

Environment: AIVAULT_VERSION, AIVAULT_INSTALL_DIR, AIVAULT_HOME.
USAGE
}
while [ "$#" -gt 0 ]; do
    case "$1" in
        --auto-connect) auto_connect=1; shift ;;
        --version|--install-dir|--vault)
            [ "$#" -ge 2 ] || fail "Missing value for $1"
            case "$1" in
                --version) version=$2 ;;
                --install-dir) install_dir=$2 ;;
                --vault) vault_dir=$2 ;;
            esac
            shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) fail "Unknown option: $1 (see --help)" ;;
    esac
done

case "$(uname -s)" in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    *) fail 'Use install.ps1 on Windows. Supported systems: macOS, Linux, Windows.' ;;
esac
case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) fail 'Supported CPUs: x86-64 and ARM64.' ;;
esac
command -v tar >/dev/null 2>&1 || fail 'tar is required.'
if command -v sha256sum >/dev/null 2>&1; then
    hash_tool=sha256sum
elif command -v shasum >/dev/null 2>&1; then
    hash_tool=shasum
elif command -v openssl >/dev/null 2>&1; then
    hash_tool=openssl
else
    fail 'A SHA-256 tool is required: sha256sum, shasum, or openssl.'
fi

# The only download override is an explicit local test fixture, never a mirror URL.
fixture=${AIVAULT_TEST_RELEASE_DIR:-}
if [ -n "$fixture" ]; then
    case "$fixture" in /*) ;; *) fail 'AIVAULT_TEST_RELEASE_DIR must be an absolute directory.' ;; esac
    [ -d "$fixture" ] || fail 'Release fixture directory does not exist.'
    [ -n "$version" ] || fail 'A local release fixture requires --version or AIVAULT_VERSION.'
else
    command -v curl >/dev/null 2>&1 || fail 'curl is required.'
    if [ -z "$version" ]; then
        latest_url=$(curl --proto '=https' --proto-redir '=https' --tlsv1.2 -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")
        case "$latest_url" in
            "https://github.com/$repo/releases/tag/"*) version=${latest_url##*/} ;;
            *) fail 'Could not resolve the latest release. Use --version v0.1.0.' ;;
        esac
    fi
fi
printf '%s\n' "$version" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$' || fail 'Invalid release version.'
version=${version#v}
tag=v$version
asset=aivault_${version}_${os}_${arch}.tar.gz
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/aivault-install.XXXXXX")
cleanup() {
    [ -z "$staged_binary" ] || rm -f "$staged_binary"
    [ -z "$tmp_dir" ] || rm -rf "$tmp_dir"
}
trap cleanup EXIT
trap 'exit 130' HUP INT TERM
fetch() {
    if [ -n "$fixture" ]; then
        cp "$fixture/$1" "$tmp_dir/$1"
    else
        curl --proto '=https' --proto-redir '=https' --tlsv1.2 -fsSL --retry 3 \
            "https://github.com/$repo/releases/download/$tag/$1" -o "$tmp_dir/$1"
    fi
}
printf 'Installing AI Vault %s for %s/%s\n' "$tag" "$os" "$arch"
fetch "$asset"
fetch SHA256SUMS
if ! expected=$(awk -v name="$asset" '$2 == name || $2 == "*" name { count++; value=$1 } END { if (count != 1) exit 1; print value }' "$tmp_dir/SHA256SUMS"); then
    fail 'Release checksum entry missing or duplicated.'
fi
printf '%s\n' "$expected" | grep -Eq '^[0-9a-fA-F]{64}$' || fail 'Release checksum entry missing, invalid, or duplicated.'
case "$hash_tool" in
    sha256sum) actual=$(sha256sum "$tmp_dir/$asset" | awk '{print $1}') ;;
    shasum) actual=$(shasum -a 256 "$tmp_dir/$asset" | awk '{print $1}') ;;
    openssl) actual=$(openssl dgst -sha256 "$tmp_dir/$asset" | awk '{print $NF}') ;;
esac
expected=$(printf '%s' "$expected" | tr 'A-F' 'a-f')
[ "$actual" = "$expected" ] || fail 'SHA-256 mismatch. Nothing was installed.'
# Stream only the expected root binary to a file we create; archive paths and
# symlinks are never materialized on disk.
tar -xOzf "$tmp_dir/$asset" aivault > "$tmp_dir/aivault"
[ -s "$tmp_dir/aivault" ] || fail 'Archive does not contain an aivault binary.'
chmod 755 "$tmp_dir/aivault"
"$tmp_dir/aivault" version >/dev/null
[ -n "$install_dir" ] && [ -n "$vault_dir" ] || fail 'Install and vault directories must not be empty.'
mkdir -p "$install_dir" "$vault_dir"
install_dir=$(cd "$install_dir" && pwd -P)
vault_dir=$(cd "$vault_dir" && pwd -P)
staged_binary=$(mktemp "$install_dir/.aivault.XXXXXX")
cp "$tmp_dir/aivault" "$staged_binary"
chmod 755 "$staged_binary"
mv -f "$staged_binary" "$install_dir/aivault"
staged_binary=
binary=$install_dir/aivault
"$binary" --vault "$vault_dir" init >/dev/null
"$binary" version
printf '\nExecutable: %s\nShared vault: %s\n' "$binary" "$vault_dir"

connect_failed=0
if [ "$auto_connect" -eq 1 ]; then
    if command -v codex >/dev/null 2>&1; then
        if codex mcp get ai-vault >/dev/null 2>&1; then
            printf 'Codex: existing ai-vault entry preserved; inspect with codex mcp get ai-vault.\n'
        elif codex mcp add ai-vault -- "$binary" --vault "$vault_dir" serve; then
            printf 'Codex: ai-vault registered.\n'
        else
            printf 'Codex registration failed; use the configuration below.\n' >&2
            connect_failed=1
        fi
    else
        printf 'Codex CLI not found; automatic registration skipped.\n'
    fi
    if command -v claude >/dev/null 2>&1; then
        if claude mcp get ai-vault >/dev/null 2>&1; then
            printf 'Claude Code: existing ai-vault entry preserved; inspect with claude mcp get ai-vault.\n'
        elif claude mcp add --scope user --transport stdio ai-vault -- "$binary" --vault "$vault_dir" serve; then
            printf 'Claude Code: ai-vault registered for this user.\n'
        else
            printf 'Claude Code registration failed; use the configuration below.\n' >&2
            connect_failed=1
        fi
    else
        printf 'Claude Code CLI not found; automatic registration skipped.\n'
    fi
fi
printf '\nMCP JSON for Claude Desktop and other local stdio clients:\n'
"$binary" --vault "$vault_dir" config --format json
printf '\nRestart or reconnect your AI client, then ask it to list AI Vault notes.\n'
printf 'Use the absolute executable path above, or add %s to your PATH.\n' "$install_dir"
[ "$connect_failed" -eq 0 ] || fail 'Installation succeeded, but a client registration failed.'
