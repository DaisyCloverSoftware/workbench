param(
    [Parameter(Mandatory = $true)]
    [string]$AppId,
    [string]$Destination = (Join-Path $HOME ".codex\plugins\workbench"),
    [string]$Marketplace = (Join-Path $HOME ".agents\plugins\marketplace.json")
)

$ErrorActionPreference = "Stop"
if ($AppId -notmatch '^plugin_asdk_app_[A-Za-z0-9_-]+$') {
    throw "AppId must be the technical plugin_asdk_app... id of the registered Workbench MCP connection."
}

$Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
function Write-Utf8NoBom([string]$Path, [string]$Text) {
    [IO.File]::WriteAllText($Path, $Text, $script:Utf8NoBom)
}

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$SourceManifest = Join-Path $RepoRoot ".codex-plugin\plugin.json"
$SourceSkill = Join-Path $RepoRoot "skills\workbench"
$Parent = Split-Path -Parent $Destination
New-Item -ItemType Directory -Force -Path $Parent | Out-Null
$Temp = Join-Path $Parent (".workbench-plugin." + [Guid]::NewGuid().ToString("N"))

try {
    New-Item -ItemType Directory -Force -Path (Join-Path $Temp ".codex-plugin"), (Join-Path $Temp "skills") | Out-Null
    Copy-Item -Recurse -Force $SourceSkill (Join-Path $Temp "skills\workbench")

    $Manifest = Get-Content -Raw $SourceManifest | ConvertFrom-Json
    if ($Manifest.PSObject.Properties.Name -contains "apps") {
        $Manifest.apps = "./.app.json"
    } else {
        $Manifest | Add-Member -NotePropertyName apps -NotePropertyValue "./.app.json"
    }
    Write-Utf8NoBom (Join-Path $Temp ".codex-plugin\plugin.json") (($Manifest | ConvertTo-Json -Depth 20) + "`n")

    $App = [PSCustomObject]@{ apps = [PSCustomObject]@{ workbench = [PSCustomObject]@{ id = $AppId } } }
    Write-Utf8NoBom (Join-Path $Temp ".app.json") (($App | ConvertTo-Json -Depth 10) + "`n")

    Write-Utf8NoBom (Join-Path $Temp "README.md") @"
# Workbench personal ChatGPT plugin

This directory is generated from DaisyCloverSoftware/workbench. Its .app.json contains the workspace-specific technical id of the registered Workbench MCP connection. Regenerate it from the source repository rather than editing it by hand.
"@

    if (Test-Path $Destination) {
        $ExistingManifest = Join-Path $Destination ".codex-plugin\plugin.json"
        if (-not (Test-Path $ExistingManifest)) { throw "Refusing to replace non-Workbench destination: $Destination" }
        $Existing = Get-Content -Raw $ExistingManifest | ConvertFrom-Json
        if ($Existing.name -ne "workbench") { throw "Refusing to replace non-Workbench destination: $Destination" }
        $Old = Join-Path $Parent (".workbench-plugin.previous." + [Guid]::NewGuid().ToString("N"))
        Move-Item $Destination $Old
        try { Move-Item $Temp $Destination } catch { Move-Item $Old $Destination; throw }
        Remove-Item -Recurse -Force $Old
    } else {
        Move-Item $Temp $Destination
    }

    $StandardDestination = Join-Path $HOME ".codex\plugins\workbench"
    if ([IO.Path]::GetFullPath($Destination) -eq [IO.Path]::GetFullPath($StandardDestination)) {
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Marketplace) | Out-Null
        if (Test-Path $Marketplace) {
            $Market = Get-Content -Raw $Marketplace | ConvertFrom-Json
        } else {
            $Market = [PSCustomObject]@{
                name = "personal"
                interface = [PSCustomObject]@{ displayName = "Personal" }
                plugins = @()
            }
        }
        if (-not ($Market.PSObject.Properties.Name -contains "plugins")) {
            $Market | Add-Member -NotePropertyName plugins -NotePropertyValue @()
        }
        $Entry = [PSCustomObject]@{
            name = "workbench"
            source = [PSCustomObject]@{ source = "local"; path = "./.codex/plugins/workbench" }
            policy = [PSCustomObject]@{ installation = "AVAILABLE"; authentication = "ON_INSTALL" }
            category = "Developer Tools"
        }
        $Plugins = @($Market.plugins)
        $Replaced = $false
        for ($i = 0; $i -lt $Plugins.Count; $i++) {
            if ($Plugins[$i].name -eq "workbench") { $Plugins[$i] = $Entry; $Replaced = $true; break }
        }
        if (-not $Replaced) { $Plugins += $Entry }
        $Market.plugins = $Plugins
        Write-Utf8NoBom $Marketplace (($Market | ConvertTo-Json -Depth 20) + "`n")
    }

    Write-Host "WORKBENCH CHATGPT PLUGIN PACKAGE READY"
    Write-Host "  plugin: $Destination"
    Write-Host "  app binding: workspace-specific id written locally"
    if ([IO.Path]::GetFullPath($Destination) -eq [IO.Path]::GetFullPath($StandardDestination)) {
        Write-Host "  marketplace: $Marketplace"
    }
} finally {
    if (Test-Path $Temp) { Remove-Item -Recurse -Force $Temp }
}
