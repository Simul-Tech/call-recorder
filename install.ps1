#Requires -Version 5
$ErrorActionPreference = "Stop"

$Repo    = "Simul-Tech/call-recorder"
$Binary  = "call-recorder"
$InstDir = "$env:LOCALAPPDATA\Programs\call-recorder"
$Asset   = "call-recorder-windows-amd64.exe"

# ── Find latest release ───────────────────────────────────────────────────────

$ApiUrl = "https://api.github.com/repos/$Repo/releases/latest"

try {
    $Release = Invoke-RestMethod -Uri $ApiUrl
    $Latest  = $Release.tag_name
} catch {
    Write-Host "Impossibile recuperare l'ultima release da GitHub."
    exit 1
}

Write-Host "Ultima release: $Latest"

# ── Download ──────────────────────────────────────────────────────────────────

$DownloadUrl = "https://github.com/$Repo/releases/download/$Latest/$Asset"
New-Item -ItemType Directory -Force -Path $InstDir | Out-Null
$Dest = "$InstDir\$Binary.exe"

Write-Host "Scaricando $Asset..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $Dest

# ── Add to PATH ───────────────────────────────────────────────────────────────

$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstDir", "User")
    Write-Host "Aggiunto al PATH: $InstDir (riapri il terminale per attivarlo)"
}

Write-Host ""
Write-Host "✓ call-recorder $Latest installato in $Dest"
Write-Host ""
Write-Host "Utilizzo:"
Write-Host "  call-recorder list"
Write-Host "  call-recorder record -lang it"
Write-Host "  call-recorder tray -lang it"
