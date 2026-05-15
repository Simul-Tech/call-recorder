#Requires -Version 5
$ErrorActionPreference = "Stop"

$Gitlab  = "https://gitlab.simultech.it"
$Project = "simultech/call-recorder"
$Binary  = "call-recorder"
$InstDir = "$env:LOCALAPPDATA\Programs\call-recorder"
$Asset   = "call-recorder-windows-amd64.exe"

# ── Auth ──────────────────────────────────────────────────────────────────────

$Token = $env:GITLAB_TOKEN
$Headers = @{}
if ($Token) { $Headers["PRIVATE-TOKEN"] = $Token }

# ── Find latest release ───────────────────────────────────────────────────────

$ProjectEncoded = [Uri]::EscapeDataString($Project)
$ApiUrl = "$Gitlab/api/v4/projects/$ProjectEncoded/releases"

try {
    $Releases = Invoke-RestMethod -Uri $ApiUrl -Headers $Headers
    $Latest = $Releases[0].tag_name
} catch {
    Write-Host ""
    Write-Host "Impossibile recuperare la lista release."
    if (-not $Token) {
        Write-Host ""
        Write-Host "Il repository e' privato. Genera un Personal Access Token su:"
        Write-Host "  $Gitlab/-/user_settings/personal_access_tokens"
        Write-Host "(scope: read_api)"
        Write-Host ""
        Write-Host "Poi esegui:"
        Write-Host "  `$env:GITLAB_TOKEN='<token>'; irm .../install.ps1 | iex"
    }
    exit 1
}

Write-Host "Ultima release: $Latest"

# ── Download ──────────────────────────────────────────────────────────────────

$PkgUrl = "$Gitlab/api/v4/projects/$ProjectEncoded/packages/generic/$Binary/$Latest/$Asset"
New-Item -ItemType Directory -Force -Path $InstDir | Out-Null
$Dest = "$InstDir\$Binary.exe"

Write-Host "Scaricando $Asset..."
Invoke-WebRequest -Uri $PkgUrl -Headers $Headers -OutFile $Dest

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
