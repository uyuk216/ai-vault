#!/usr/bin/env python3
"""Run the real native installer against an isolated, checksum-verified release.

Only Python's standard library, the native Go toolchain and the installer's normal
platform prerequisites are used. No network downloads or real AI client settings
are involved. Windows deliberately tests Windows PowerShell 5.1, not just pwsh.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import shlex
import subprocess
import sys
import tarfile
import tempfile
import zipfile


ROOT = Path(__file__).resolve().parent.parent
VERSION = "0.0.0-test"


def run(command: list[str | Path], env: dict[str, str], *, succeeds: bool = True) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        [str(item) for item in command], cwd=ROOT, env=env,
        text=True, encoding="utf-8", errors="replace", capture_output=True,
        timeout=180,
    )
    if (result.returncode == 0) != succeeds:
        raise AssertionError(
            f"Unexpected exit {result.returncode} from {command!r}\n"
            f"stdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        )
    return result


def client_stubs(directory: Path, env: dict[str, str], binary: Path, vault: Path) -> Path:
    stubs = directory / "Client stubs with spaces"
    stubs.mkdir()
    state = directory / "Stub state"
    state.mkdir()
    log = directory / "client-calls.jsonl"
    helper = stubs / "client_stub.py"
    helper.write_text(r'''import json
import os
from pathlib import Path
import sys

client, args = sys.argv[1], sys.argv[2:]
with open(os.environ["AIVAULT_TEST_CLIENT_LOG"], "a", encoding="utf-8") as f:
    f.write(json.dumps({"client": client, "args": args}) + "\n")
state = Path(os.environ["AIVAULT_TEST_CLIENT_STATE"]) / client
if args == ["mcp", "get", "ai-vault"]:
    sys.exit(0 if state.exists() else 1)
if client == "codex":
    expected = ["mcp", "add", "ai-vault", "--"]
else:
    expected = ["mcp", "add", "--scope", "user", "--transport", "stdio", "ai-vault", "--"]
expected += [os.environ["AIVAULT_TEST_EXPECT_BINARY"], "--vault", os.environ["AIVAULT_TEST_EXPECT_VAULT"], "serve"]
if args != expected:
    print("Unexpected client registration arguments: " + repr(args), file=sys.stderr)
    sys.exit(2)
state.write_text("registered", encoding="utf-8")
''', encoding="utf-8")
    for client in ("codex", "claude"):
        if os.name == "nt":
            launcher = stubs / (client + ".cmd")
            launcher.write_text(
                f'@"{sys.executable}" "{helper}" {client} %*\n@exit /b %errorlevel%\n',
                encoding="utf-8",
            )
        else:
            launcher = stubs / client
            launcher.write_text(
                f"#!/bin/sh\nexec {shlex.quote(sys.executable)} {shlex.quote(str(helper))} {client} \"$@\"\n",
                encoding="utf-8",
            )
            launcher.chmod(0o755)
    env.update({
        "PATH": str(stubs) + os.pathsep + env.get("PATH", ""),
        "AIVAULT_TEST_CLIENT_LOG": str(log),
        "AIVAULT_TEST_CLIENT_STATE": str(state),
        "AIVAULT_TEST_EXPECT_BINARY": str(binary),
        "AIVAULT_TEST_EXPECT_VAULT": str(vault),
    })
    return log


def make_release(fixture: Path, binary: Path, target_os: str, target_arch: str) -> Path:
    suffix = ".zip" if target_os == "windows" else ".tar.gz"
    asset = fixture / f"aivault_{VERSION}_{target_os}_{target_arch}{suffix}"
    source_files = [
        (binary, binary.name),
        (ROOT / "LICENSE", "LICENSE"),
        *((readme, readme.name) for readme in sorted(ROOT.glob("README*.md"))),
        (ROOT / "THIRD_PARTY_NOTICES.md", "THIRD_PARTY_NOTICES.md"),
        (ROOT / "third_party_licenses", "third_party_licenses"),
    ]
    if target_os == "windows":
        with zipfile.ZipFile(asset, "w", zipfile.ZIP_DEFLATED) as archive:
            for source, name in source_files:
                if source.is_dir():
                    for item in sorted(source.rglob("*")):
                        if item.is_file():
                            archive.write(item, str(Path(name) / item.relative_to(source)))
                else:
                    archive.write(source, name)
    else:
        with tarfile.open(asset, "w:gz") as archive:
            for source, name in source_files:
                archive.add(source, arcname=name)
    digest = hashlib.sha256(asset.read_bytes()).hexdigest()
    (fixture / "SHA256SUMS").write_text(f"{digest}  {asset.name}\n", encoding="ascii")
    return asset


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--go", default=os.environ.get("GO_BINARY", "go"), help="native Go executable")
    arguments = parser.parse_args()
    target_os = {"Darwin": "darwin", "Linux": "linux", "Windows": "windows"}.get(platform.system())
    target_arch = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(platform.machine().lower())
    if target_os is None or target_arch is None:
        raise SystemExit("Installer smoke test requires a supported native OS and architecture")

    with tempfile.TemporaryDirectory(prefix="AI Vault installer smoke ") as temporary:
        directory = Path(temporary).resolve()
        fixture = directory / "Release fixture"
        fixture.mkdir()
        build = directory / "Build output"
        build.mkdir()
        name = "aivault.exe" if os.name == "nt" else "aivault"
        built_binary = build / name
        install_dir = directory / "Install directory with spaces"
        binary = install_dir / name
        vault = directory / "Shared vault with spaces"
        home = directory / "Isolated home"
        home.mkdir()
        env = os.environ.copy()
        for key in ("GOOS", "GOARCH", "AIVAULT_VERSION", "AIVAULT_INSTALL_DIR", "AIVAULT_HOME", "CODEX_HOME", "CLAUDE_CONFIG_DIR"):
            env.pop(key, None)
        env.update({"CGO_ENABLED": "0", "PYTHONUTF8": "1"})
        # Build before isolating HOME so normal pre-populated Go caches still work.
        run([arguments.go, "build", "-trimpath", f"-ldflags=-s -w -X main.version={VERSION}", "-o", built_binary, "./cmd/aivault"], env)
        asset = make_release(fixture, built_binary, target_os, target_arch)
        env.update({
            "HOME": str(home), "USERPROFILE": str(home),
            "APPDATA": str(home / "AppData" / "Roaming"),
            "LOCALAPPDATA": str(home / "AppData" / "Local"),
            "CODEX_HOME": str(home / ".codex"),
            "CLAUDE_CONFIG_DIR": str(home / ".claude"),
            "AIVAULT_TEST_RELEASE_DIR": str(fixture),
        })
        log = client_stubs(directory, env, binary, vault)

        def installer(install: Path, store: Path) -> list[str | Path]:
            if os.name == "nt":
                return [
                    "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
                    "-File", ROOT / "install.ps1", "-AutoConnect", "-Version", "v" + VERSION,
                    "-InstallDir", install, "-Vault", store,
                ]
            return ["sh", ROOT / "install.sh", "--auto-connect", "--version", "v" + VERSION, "--install-dir", install, "--vault", store]

        if os.name == "nt":
            env["AIVAULT_TEST_INSTALLER"] = str(ROOT / "install.ps1")
            parsed = run([
                "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
                "$tokens=$null; $errors=$null; [System.Management.Automation.Language.Parser]::ParseFile($env:AIVAULT_TEST_INSTALLER,[ref]$tokens,[ref]$errors) | Out-Null; if($errors.Count){$errors | Out-String | Write-Error; exit 1}; $PSVersionTable.PSVersion.ToString()",
            ], env)
            print("PowerShell parser/runtime version:", parsed.stdout.strip())
        else:
            run(["sh", "-n", ROOT / "install.sh"], env)

        run(installer(install_dir, vault), env)
        assert binary.is_file(), "Installer did not install the release binary"

        def cli(*args: str) -> object:
            return json.loads(run([binary, "--vault", vault, *args], env).stdout)

        assert cli("version")["version"] == VERSION
        assert cli("list") == [], "Smoke test vault was not empty"
        content = "# Installer smoke test\nShared memory #test\n"
        note = cli("write", "note.md", "--category", "Smoke tests/日本語", "--content", content)
        note_path = "Smoke tests/日本語/note.md"
        assert note["path"] == note_path
        assert cli("read", note_path)["content"] == content
        categories = cli("categories")
        assert {item["path"]: item["notes"] for item in categories} == {"Smoke tests": 1, "Smoke tests/日本語": 1}
        updated = content + "Saved by another AI client\n"
        cli("write", note_path, "--content", updated, "--expected-sha", note["sha256"])
        configuration = cli("config")["mcpServers"]["ai-vault"]
        assert Path(configuration["command"]) == binary
        assert configuration["args"] == ["--vault", str(vault), "serve"]

        calls = [json.loads(line) for line in log.read_text(encoding="utf-8").splitlines()]
        assert len(calls) == 4, f"Expected both get/add registration paths, got {calls!r}"
        assert {call["client"] for call in calls} == {"codex", "claude"}
        assert sum(call["args"][1] == "add" for call in calls) == 2
        run(installer(install_dir, vault), env)
        calls = [json.loads(line) for line in log.read_text(encoding="utf-8").splitlines()]
        assert len(calls) == 6 and sum(call["args"][1] == "add" for call in calls) == 2, "Reinstall changed existing AI client registrations"
        assert cli("read", note_path)["content"] == updated, "Reinstall changed existing notes"

        # A failed integrity check must happen before either installation or init.
        (fixture / "SHA256SUMS").write_text(f"{'0' * 64}  {asset.name}\n", encoding="ascii")
        rejected_install = directory / "Rejected install"
        rejected_vault = directory / "Rejected vault"
        rejected = run(installer(rejected_install, rejected_vault), env, succeeds=False)
        assert "mismatch" in (rejected.stdout + rejected.stderr).lower(), "Installer failed without reporting the checksum mismatch"
        assert not rejected_install.exists() and not rejected_vault.exists(), "Failed checksum created installation/vault data"
        assert cli("read", note_path)["content"] == updated
        print(f"Native {target_os}/{target_arch} installer passed: verified fixture, paths with spaces, isolated client registration, read/write/categories, preserved reinstall, corrupted checksum rejected.")


if __name__ == "__main__":
    main()
