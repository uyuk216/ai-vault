#Requires -Version 5.1
<#
AI Vault Windows release installer. Run without administrator permissions.
The binary and SHA256SUMS are always downloaded from the same release tag.
#>
[CmdletBinding()]
param(
    [switch]$AutoConnect,
    [string]$Version = $env:AIVAULT_VERSION,
    [string]$InstallDir = $env:AIVAULT_INSTALL_DIR,
    [string]$Vault = $env:AIVAULT_HOME
)
$ErrorActionPreference = 'Stop'
$repo = 'uyuk216/ai-vault'
$temporaryDirectory = $null
$stagedBinary = $null
try {
    if ($env:OS -ne 'Windows_NT') { throw 'Use install.sh on macOS and Linux.' }
    if ([string]::IsNullOrWhiteSpace($InstallDir)) { $InstallDir = Join-Path $env:LOCALAPPDATA 'AI-Vault\bin' }
    if ([string]::IsNullOrWhiteSpace($Vault)) { $Vault = Join-Path ([Environment]::GetFolderPath('UserProfile')) 'AI-Vault' }
    $cpu = $env:PROCESSOR_ARCHITEW6432
    if ([string]::IsNullOrWhiteSpace($cpu)) { $cpu = $env:PROCESSOR_ARCHITECTURE }
    switch ($cpu.ToUpperInvariant()) {
        'AMD64' { $architecture = 'amd64' }
        'ARM64' { $architecture = 'arm64' }
        default { throw 'Supported CPUs: x86-64 and ARM64.' }
    }
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    # Test fixture override is local-only and requires an explicit release version.
    $fixture = $env:AIVAULT_TEST_RELEASE_DIR
    if (-not [string]::IsNullOrWhiteSpace($fixture)) {
        if (-not [IO.Path]::IsPathRooted($fixture) -or -not (Test-Path -LiteralPath $fixture -PathType Container)) {
            throw 'AIVAULT_TEST_RELEASE_DIR must be an absolute existing directory.'
        }
        if ([string]::IsNullOrWhiteSpace($Version)) { throw 'A local release fixture requires -Version or AIVAULT_VERSION.' }
    } elseif ([string]::IsNullOrWhiteSpace($Version)) {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers @{ 'User-Agent' = 'AI-Vault-Installer' }
        $Version = $release.tag_name
    }
    if ($Version -notmatch '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$') { throw 'Invalid release version.' }
    $Version = $Version -replace '^v', ''
    $tag = "v$Version"
    $asset = "aivault_${Version}_windows_${architecture}.zip"
    $temporaryDirectory = Join-Path ([IO.Path]::GetTempPath()) ('aivault-install-' + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null
    Write-Host "Installing AI Vault $tag for windows/$architecture"
    foreach ($name in @($asset, 'SHA256SUMS')) {
        $destination = Join-Path $temporaryDirectory $name
        if (-not [string]::IsNullOrWhiteSpace($fixture)) {
            Copy-Item -LiteralPath (Join-Path $fixture $name) -Destination $destination
        } else {
            Invoke-WebRequest -UseBasicParsing -Uri "https://github.com/$repo/releases/download/$tag/$name" -OutFile $destination
        }
    }
    $checksumLines = @(Get-Content -LiteralPath (Join-Path $temporaryDirectory 'SHA256SUMS') | Where-Object {
        $_ -match ('^([0-9a-fA-F]{64})\s+\*?' + [Regex]::Escape($asset) + '$')
    })
    if ($checksumLines.Count -ne 1) { throw 'Release checksum entry missing, invalid, or duplicated.' }
    $expected = ($checksumLines[0] -split '\s+')[0]
    $actual = (Get-FileHash -LiteralPath (Join-Path $temporaryDirectory $asset) -Algorithm SHA256).Hash
    if ($actual -ne $expected) { throw 'SHA-256 mismatch. Nothing was installed.' }
    # Read just the expected regular binary; never unpack arbitrary archive paths.
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [IO.Compression.ZipFile]::OpenRead((Join-Path $temporaryDirectory $asset))
    try {
        $entries = @($archive.Entries | Where-Object { $_.FullName -eq 'aivault.exe' })
        if ($entries.Count -ne 1 -or $entries[0].Length -eq 0) { throw 'Archive does not contain aivault.exe at its root.' }
        [IO.Compression.ZipFileExtensions]::ExtractToFile($entries[0], (Join-Path $temporaryDirectory 'aivault.exe'), $false)
    } finally { $archive.Dispose() }
    & (Join-Path $temporaryDirectory 'aivault.exe') version | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Release binary could not run on this computer.' }
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    New-Item -ItemType Directory -Path $Vault -Force | Out-Null
    $InstallDir = (Resolve-Path -LiteralPath $InstallDir).ProviderPath
    $Vault = (Resolve-Path -LiteralPath $Vault).ProviderPath
    $binary = Join-Path $InstallDir 'aivault.exe'
    $stagedBinary = Join-Path $InstallDir ('.aivault-' + [Guid]::NewGuid().ToString('N') + '.exe')
    Copy-Item -LiteralPath (Join-Path $temporaryDirectory 'aivault.exe') -Destination $stagedBinary
    Move-Item -LiteralPath $stagedBinary -Destination $binary -Force
    $stagedBinary = $null
    & $binary --vault $Vault init | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Vault initialization failed.' }
    & $binary version
    if ($LASTEXITCODE -ne 0) { throw 'Installed binary could not run.' }
    Write-Host "`nExecutable: $binary`nShared vault: $Vault"
    $connectFailed = $false
    if ($AutoConnect) {
        foreach ($client in @('codex', 'claude')) {
            if ($null -eq (Get-Command $client -ErrorAction SilentlyContinue)) {
                Write-Host "$client CLI not found; automatic registration skipped."
                continue
            }
            # Native stderr on PowerShell 5.1 is an ErrorRecord even for an expected
            # missing entry. Temporarily allow it, then check the native exit code.
            $savedPreference = $ErrorActionPreference
            try {
                $ErrorActionPreference = 'Continue'
                & $client mcp get ai-vault *> $null
                $exists = $LASTEXITCODE -eq 0
            } finally { $ErrorActionPreference = $savedPreference }
            if ($exists) {
                Write-Host "$client`: existing ai-vault entry preserved; inspect with $client mcp get ai-vault."
                continue
            }
            try {
                $ErrorActionPreference = 'Continue'
                if ($client -eq 'codex') {
                    & $client mcp add ai-vault -- $binary --vault $Vault serve
                } else {
                    & $client mcp add --scope user --transport stdio ai-vault -- $binary --vault $Vault serve
                }
                $registrationExitCode = $LASTEXITCODE
            } finally { $ErrorActionPreference = $savedPreference }
            if ($registrationExitCode -ne 0) {
                Write-Warning "$client registration failed; use the configuration below."
                $connectFailed = $true
            } else { Write-Host "$client`: ai-vault registered." }
        }
    }
    Write-Host "`nMCP JSON for Claude Desktop and other local stdio clients:"
    & $binary --vault $Vault config --format json
    if ($LASTEXITCODE -ne 0) { throw 'Could not generate MCP configuration.' }
    Write-Host "`nRestart or reconnect your AI client, then ask it to list AI Vault notes."
    Write-Host "Use the absolute executable path above, or add $InstallDir to your user PATH."
    if ($connectFailed) { throw 'Installation succeeded, but a client registration failed.' }
} finally {
    if ($stagedBinary -and (Test-Path -LiteralPath $stagedBinary)) { Remove-Item -LiteralPath $stagedBinary -Force }
    if ($temporaryDirectory -and (Test-Path -LiteralPath $temporaryDirectory)) { Remove-Item -LiteralPath $temporaryDirectory -Recurse -Force }
}
