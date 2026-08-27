# install.ps1 - obsidian-doubleclick
#
# Registers obsidian-doubleclick.exe as the program Windows runs when you
# open a .md file. Writes to HKCU only, so no administrator rights are needed.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File install.ps1
#
# Before running:
#   1. obsidian-doubleclick.exe must sit in this same folder
#   2. Set Obsidian as the default app for .md in Windows Settings FIRST.
#      Doing it afterwards lets Windows overwrite what this script registers.

$ErrorActionPreference = "Stop"

$scriptRoot     = Split-Path -Parent $MyInvocation.MyCommand.Path
$handlerExe     = Join-Path $scriptRoot "obsidian-doubleclick.exe"
$backupDir      = Join-Path $scriptRoot ".backup"
$timestamp      = Get-Date -Format "yyyyMMdd_HHmmss"
$backupPath     = Join-Path $backupDir "backup_$timestamp.json"

$registryPath   = "HKCU:\Software\Classes\Applications\Obsidian.exe\shell\open\command"
$userChoicePath = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\.md\UserChoice"

# Locate the real Obsidian install so the file icon stays correct
$obsidianCandidates = @(
  "$env:LOCALAPPDATA\Programs\Obsidian\Obsidian.exe",
  "C:\Program Files\Obsidian\Obsidian.exe"
)
$obsidianExePath = $obsidianCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $obsidianExePath) {
  $obsidianExePath = "C:\Program Files\Obsidian\Obsidian.exe"  # fallback, icon only
}

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------

if (-not (Test-Path $handlerExe)) {
  Write-Error "Cannot find obsidian-doubleclick.exe at: $handlerExe"
  Write-Error "Download it from the Releases page into this folder, or build it:"
  Write-Error "  go build -ldflags '-H=windowsgui' -o obsidian-doubleclick.exe ."
  exit 1
}

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

function Set-RegistryDefaultValue {
  param([string]$Path, [string]$Value)
  $subKeyPath = $Path -replace '^HKCU:\\', ''
  $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($subKeyPath, $true)
  if (-not $key) { throw "Cannot open registry key for writing: $Path" }
  try { $key.SetValue("", $Value, [Microsoft.Win32.RegistryValueKind]::String) }
  finally { $key.Dispose() }
}

# ---------------------------------------------------------------------------
# Back up whatever is there now
# ---------------------------------------------------------------------------

New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
New-Item -Path $registryPath -Force | Out-Null

$previousCommand = (Get-Item -Path $registryPath).GetValue("")
$currentProgId   = if (Test-Path $userChoicePath) {
  (Get-ItemProperty -Path $userChoicePath).ProgId
} else { $null }

@{
  createdAt        = (Get-Date).ToString("o")
  registryPath     = $registryPath
  previousCommand  = $previousCommand
  userChoiceProgId = $currentProgId
} | ConvertTo-Json | Set-Content -Path $backupPath -Encoding UTF8

# ---------------------------------------------------------------------------
# Register
# ---------------------------------------------------------------------------

$newCommand = """$handlerExe"" ""%1"""
Set-RegistryDefaultValue -Path $registryPath -Value $newCommand
$verifiedCommand = (Get-Item -Path $registryPath).GetValue("")

# Keep the Obsidian icon on .md files
$iconKeyPath = "HKCU:\Software\Classes\Applications\Obsidian.exe\DefaultIcon"
New-Item -Path $iconKeyPath -Force | Out-Null
Set-RegistryDefaultValue -Path $iconKeyPath -Value """$obsidianExePath"",0"
$verifiedIcon = (Get-Item -Path $iconKeyPath).GetValue("")

# Keep the app named "Obsidian" in the Open With menu
$appKey = "HKCU:\Software\Classes\Applications\Obsidian.exe"
Set-ItemProperty -Path $appKey -Name "FriendlyAppName" -Value "Obsidian"
$verifiedAppName = (Get-ItemProperty -Path $appKey).FriendlyAppName

# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------

Write-Output ""
Write-Output "[OK] obsidian-doubleclick installed"
Write-Output ""
Write-Output "  Handler exe  : $handlerExe"
Write-Output "  Command      : $verifiedCommand"
Write-Output "  DefaultIcon  : $verifiedIcon"
Write-Output "  AppName      : $verifiedAppName"
Write-Output "  Backup       : $backupPath"
Write-Output "  .md ProgId   : $currentProgId"
Write-Output ""

if ($currentProgId -ne "Applications\Obsidian.exe") {
  Write-Output "[!] The default app for .md is not 'Applications\Obsidian.exe' yet."
  Write-Output "    Open Windows Settings > Apps > Default apps, search for .md,"
  Write-Output "    choose Obsidian, then run this script again."
  Write-Output "    (Windows can overwrite the command if you do it in the other order.)"
} else {
  Write-Output "[OK] Default app for .md is Applications\Obsidian.exe - ready."
  Write-Output "     Double-click a .md file in Explorer and it will open in Obsidian."
}
