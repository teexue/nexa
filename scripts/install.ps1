# Download Nexa into the current directory (Windows).
# 一键下载 Nexa 到当前目录（Windows）
#
# Usage / 用法 (PowerShell):
#   irm https://raw.githubusercontent.com/teexue/nexa/main/scripts/install.ps1 | iex
#   或：powershell -ExecutionPolicy Bypass -File .\install.ps1
[CmdletBinding()]
param(
    [string]$Repo = $(if ($env:NEXA_REPO) { $env:NEXA_REPO } else { "teexue/nexa" })
)

$ErrorActionPreference = "Stop"
$OutName = "nexa.exe"

function Resolve-UiLang {
    $culture = [System.Globalization.CultureInfo]::CurrentUICulture.Name
    if ($culture -match '^(?i)zh') { return "zh" }
    return "en"
}

$script:UiLang = Resolve-UiLang

function T {
    param(
        [Parameter(Mandatory = $true)][string]$Key,
        [Parameter(ValueFromRemainingArguments = $true)][object[]]$FormatArgs
    )
    $catalog = @{
        en = @{
            err_arch    = "Unsupported CPU architecture: {0} (64-bit Windows required)"
            downloading = "Downloading {0} …"
            saved       = "Saved to:  {0}"
            run         = "Run:       .\{0}"
            browser     = "Browser:   http://localhost:8080"
        }
        zh = @{
            err_arch    = "不支持的 CPU 架构: {0}（需要 64 位 Windows）"
            downloading = "正在下载 {0} …"
            saved       = "已下载到: {0}"
            run         = "启动:      .\{0}"
            browser     = "浏览器:    http://localhost:8080"
        }
    }
    $msg = $catalog[$script:UiLang][$Key]
    if (-not $msg) { $msg = $Key }
    if ($FormatArgs -and $FormatArgs.Count -gt 0) {
        return ($msg -f $FormatArgs)
    }
    return $msg
}

function Get-CpuArch {
    $a = $env:PROCESSOR_ARCHITECTURE
    if ($a -match "ARM64") { return "arm64" }
    if ([Environment]::Is64BitOperatingSystem) { return "amd64" }
    throw (T err_arch $a)
}

$arch = Get-CpuArch
$asset = "nexa-windows-$arch.exe"
$url = "https://github.com/$Repo/releases/latest/download/$asset"
$dest = Join-Path (Get-Location) $OutName

Write-Host (T downloading $asset)
Write-Host "  $url"

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("nexa-" + [guid]::NewGuid().ToString("n") + ".exe")
try {
    Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
    Move-Item -Force -Path $tmp -Destination $dest
}
finally {
    if (Test-Path -LiteralPath $tmp) {
        Remove-Item -Force -LiteralPath $tmp -ErrorAction SilentlyContinue
    }
}

Write-Host ""
Write-Host (T saved $dest)
Write-Host (T run $OutName)
Write-Host (T browser)
