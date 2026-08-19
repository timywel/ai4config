# cfg4ai 安装脚本（Windows PowerShell）
# 用法：irm https://raw.githubusercontent.com/timywel/ai4config/main/scripts/install.ps1 | iex
param(
  [string]$Version = "latest",
  [string]$Dest = "$env:LOCALAPPDATA\Programs\cfg4ai"
)
$ErrorActionPreference = "Stop"
$Repo = "timywel/ai4config"

if ($Version -eq "latest") {
  $rel = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
  $Version = $rel.tag_name -replace '^v', ''
}

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$url = "https://github.com/$Repo/releases/download/v$Version/cfg4ai_${Version}_windows_${arch}.zip"
$tmp = Join-Path $env:TEMP "cfg4ai-install-$(Get-Random)"
New-Item -ItemType Directory -Force -Path $tmp | Out-Null

Write-Host "下载 cfg4ai v$Version (windows/$arch)..."
Invoke-WebRequest $url -OutFile (Join-Path $tmp "cfg4ai.zip")
Expand-Archive (Join-Path $tmp "cfg4ai.zip") -DestinationPath $tmp -Force

New-Item -ItemType Directory -Force -Path $Dest | Out-Null
Copy-Item (Join-Path $tmp "cfg4ai.exe") -Destination (Join-Path $Dest "cfg4ai.exe") -Force

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$Dest*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$Dest", "User")
  Write-Host "已加入用户 PATH（重开终端生效）"
}
Remove-Item -Recurse -Force $tmp
Write-Host "已安装：$Dest\cfg4ai.exe"
& (Join-Path $Dest "cfg4ai.exe") version