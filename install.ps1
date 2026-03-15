# install.ps1 — obsidian-md-handler (Go) 설치 스크립트
# 역할: HKCU\...\Applications\Obsidian.exe\shell\open\command 에
#       Go exe를 직접 등록 (wscript/node 불필요)
#
# 사용법:
#   powershell -ExecutionPolicy Bypass -File install.ps1
#
# 전제조건:
#   1. obsidian-handler.exe 가 이 스크립트와 같은 폴더에 존재할 것
#   2. Windows 설정 > 기본 앱 > .md 파일을 "Obsidian"으로 설정할 것
#      (이미 Applications\Obsidian.exe ProgId 이면 OK)

$ErrorActionPreference = "Stop"

$scriptRoot    = Split-Path -Parent $MyInvocation.MyCommand.Path
$handlerExe    = Join-Path $scriptRoot "obsidian-handler.exe"
$backupDir     = Join-Path $scriptRoot ".backup"
$timestamp     = Get-Date -Format "yyyyMMdd_HHmmss"
$backupPath    = Join-Path $backupDir "backup_$timestamp.json"

$registryPath  = "HKCU:\Software\Classes\Applications\Obsidian.exe\shell\open\command"
$userChoicePath = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\.md\UserChoice"

# Obsidian 실제 설치 경로 자동 탐색
$obsidianCandidates = @(
  "$env:LOCALAPPDATA\Programs\Obsidian\Obsidian.exe",
  "C:\Program Files\Obsidian\Obsidian.exe"
)
$obsidianExePath = $obsidianCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $obsidianExePath) {
  $obsidianExePath = "C:\Program Files\Obsidian\Obsidian.exe"  # 기본값 (아이콘용)
}

# ---------------------------------------------------------------------------
# 사전 검사
# ---------------------------------------------------------------------------

if (-not (Test-Path $handlerExe)) {
  Write-Error "obsidian-handler.exe 를 찾을 수 없습니다: $handlerExe"
  Write-Error "먼저 빌드하세요: go build -ldflags '-H=windowsgui' -o obsidian-handler.exe ."
  exit 1
}

# ---------------------------------------------------------------------------
# 헬퍼
# ---------------------------------------------------------------------------

function Set-RegistryDefaultValue {
  param([string]$Path, [string]$Value)
  $subKeyPath = $Path -replace '^HKCU:\\', ''
  $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($subKeyPath, $true)
  if (-not $key) { throw "레지스트리 키 쓰기 불가: $Path" }
  try { $key.SetValue("", $Value, [Microsoft.Win32.RegistryValueKind]::String) }
  finally { $key.Dispose() }
}

# ---------------------------------------------------------------------------
# 현재 상태 백업
# ---------------------------------------------------------------------------

New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
New-Item -Path $registryPath -Force | Out-Null

$previousCommand = (Get-Item -Path $registryPath).GetValue("")
$currentProgId   = if (Test-Path $userChoicePath) {
  (Get-ItemProperty -Path $userChoicePath).ProgId
} else { $null }

@{
  createdAt       = (Get-Date).ToString("o")
  registryPath    = $registryPath
  previousCommand = $previousCommand
  userChoiceProgId = $currentProgId
} | ConvertTo-Json | Set-Content -Path $backupPath -Encoding UTF8

# ---------------------------------------------------------------------------
# 레지스트리 등록 — Go exe 직접 연결 (wscript/node 불필요)
# ---------------------------------------------------------------------------

$newCommand = """$handlerExe"" ""%1"""
Set-RegistryDefaultValue -Path $registryPath -Value $newCommand
$verifiedCommand = (Get-Item -Path $registryPath).GetValue("")

# DefaultIcon: Obsidian 아이콘 유지
$iconKeyPath = "HKCU:\Software\Classes\Applications\Obsidian.exe\DefaultIcon"
New-Item -Path $iconKeyPath -Force | Out-Null
Set-RegistryDefaultValue -Path $iconKeyPath -Value """$obsidianExePath"",0"
$verifiedIcon = (Get-Item -Path $iconKeyPath).GetValue("")

# FriendlyAppName: 앱이름 "Obsidian"으로 표시
$appKey = "HKCU:\Software\Classes\Applications\Obsidian.exe"
Set-ItemProperty -Path $appKey -Name "FriendlyAppName" -Value "Obsidian"
$verifiedAppName = (Get-ItemProperty -Path $appKey).FriendlyAppName

# ---------------------------------------------------------------------------
# 결과 출력
# ---------------------------------------------------------------------------

Write-Output ""
Write-Output "✅ obsidian-md-handler (Go) 설치 완료"
Write-Output ""
Write-Output "  Handler exe  : $handlerExe"
Write-Output "  Command      : $verifiedCommand"
Write-Output "  DefaultIcon  : $verifiedIcon"
Write-Output "  AppName      : $verifiedAppName"
Write-Output "  Backup       : $backupPath"
Write-Output "  .md ProgId   : $currentProgId"
Write-Output ""

if ($currentProgId -ne "Applications\Obsidian.exe") {
  Write-Output "⚠️  WARNING: .md 기본 앱이 아직 'Applications\Obsidian.exe' 가 아닙니다."
  Write-Output "   Windows 설정 > 기본 앱 > '.md 파일' 을 Obsidian 으로 변경한 뒤"
  Write-Output "   이 스크립트를 다시 실행하세요 (Windows가 command를 덮어쓸 수 있음)."
} else {
  Write-Output "✅ .md 기본 앱: Applications\Obsidian.exe — 준비 완료"
  Write-Output "   Explorer 에서 .md 파일을 더블클릭하면 Go 핸들러가 실행됩니다."
}
