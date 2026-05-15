#Requires -Version 5
$ErrorActionPreference = "Stop"

$RepoUrl  = "https://gitlab.simultech.it/simultech/call-recorder"
$Binary   = "call-recorder"
$InstDir  = "$env:LOCALAPPDATA\Programs\call-recorder"
$Asset    = "call-recorder-windows-amd64.exe"

# ── Find latest release ───────────────────────────────────────────────────────

Write-Host "Recupero ultima release..."
try {
    $Html = Invoke-WebRequest -Uri "$RepoUrl/-/releases/permalink/latest" -UseBasicParsing
    $Version = [regex]::Match($Html.Content, 'v\d+\.\d+\.\d+').Value
} catch {
    $Version = $env:RELEASE
}
if (-not $Version) {
    Write-Error "Impossibile recuperare la versione. Imposta `$env:RELEASE = 'v1.0.0'"
    exit 1
}

$DownloadUrl = "$RepoUrl/-/releases/$Version/downloads/$Asset"

# ── Download ──────────────────────────────────────────────────────────────────

New-Item -ItemType Directory -Force -Path $InstDir | Out-Null
$Dest = "$InstDir\$Binary.exe"

Write-Host "Scaricando $Asset ($Version)..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $Dest

# ── Add to PATH ───────────────────────────────────────────────────────────────

$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstDir", "User")
    Write-Host "Aggiunto al PATH utente: $InstDir"
    Write-Host "Riapri il terminale per usare 'call-recorder' da qualsiasi cartella."
}

Write-Host ""
Write-Host "Installato in: $Dest"
Write-Host ""
Write-Host "Utilizzo:"
Write-Host "  call-recorder list"
Write-Host "  call-recorder record -lang it"
Write-Host "  call-recorder tray -lang it"
