<#
.SYNOPSIS
    Gera o instalador winsweep-setup.msi a partir do build de produção do Wails.

.DESCRIPTION
    1. Compila o app com `wails build`.
    2. Garante que o bootstrapper do WebView2 Runtime está baixado
       (embutido no .msi; só roda no destino se o runtime não existir lá).
    3. Compila e liga o instalador com o WiX Toolset (candle.exe / light.exe).

.PARAMETER Version
    Versão do produto no formato x.y.z (usada no .msi e no metadata do .exe).

.NOTES
    Requer o WiX Toolset v3 instalado (winget install --id WiXToolset.WiXToolset).
#>
param(
    [string]$Version = "1.0.0"
)

$ErrorActionPreference = "Stop"

$scriptDir   = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Resolve-Path (Join-Path $scriptDir "..\..\..\..")
$redistDir   = Join-Path $scriptDir "redist"
$bootstrapExe = Join-Path $redistDir "MicrosoftEdgeWebview2Setup.exe"
$objDir      = Join-Path $scriptDir "obj"
$binDir      = Join-Path $projectRoot "build\bin"
$appExe      = Join-Path $binDir "winsweep.exe"
$appIcon     = Join-Path $projectRoot "build\windows\icon.ico"
$outputMsi   = Join-Path $binDir "winsweep-setup.msi"

function Find-WixBinDir {
    $candidates = @(
        "${env:ProgramFiles(x86)}\WiX Toolset v3.14\bin",
        "${env:ProgramFiles(x86)}\WiX Toolset v3.11\bin",
        "${env:ProgramFiles}\WiX Toolset v3.14\bin"
    )
    foreach ($c in $candidates) {
        if (Test-Path (Join-Path $c "candle.exe")) { return $c }
    }
    $cmd = Get-Command candle.exe -ErrorAction SilentlyContinue
    if ($cmd) { return Split-Path $cmd.Source }
    throw "WiX Toolset não encontrado. Instale com: winget install --id WiXToolset.WiXToolset"
}

Write-Host "==> Compilando o app (wails build)..." -ForegroundColor Cyan
Push-Location $projectRoot
try {
    wails build -clean
    if ($LASTEXITCODE -ne 0) { throw "wails build falhou (exit code $LASTEXITCODE)" }
} finally {
    Pop-Location
}

if (-not (Test-Path $appExe)) {
    throw "Executável não encontrado em $appExe após o build."
}

if (-not (Test-Path $bootstrapExe)) {
    Write-Host "==> Baixando o bootstrapper do WebView2 Runtime (Evergreen, ~2MB)..." -ForegroundColor Cyan
    New-Item -ItemType Directory -Force -Path $redistDir | Out-Null
    # URL oficial e estável da Microsoft para o bootstrapper "Evergreen" do
    # WebView2 Runtime — mesmo mecanismo documentado em
    # https://learn.microsoft.com/microsoft-edge/webview2/concepts/distribution
    Invoke-WebRequest -Uri "https://go.microsoft.com/fwlink/p/?LinkId=2124703" -OutFile $bootstrapExe
}

$wixBin = Find-WixBinDir
Write-Host "==> Usando WiX Toolset em: $wixBin" -ForegroundColor Cyan

New-Item -ItemType Directory -Force -Path $objDir | Out-Null

Write-Host "==> candle.exe (compilando .wxs)..." -ForegroundColor Cyan
& "$wixBin\candle.exe" `
    -arch x64 `
    -dProductVersion="$Version" `
    -dAppExe="$appExe" `
    -dAppIcon="$appIcon" `
    -out "$objDir\" `
    "$scriptDir\winsweep.wxs"
if ($LASTEXITCODE -ne 0) { throw "candle.exe falhou (exit code $LASTEXITCODE)" }

Write-Host "==> light.exe (gerando .msi)..." -ForegroundColor Cyan
& "$wixBin\light.exe" `
    -ext WixUIExtension `
    -cultures:pt-br `
    -out "$outputMsi" `
    "$objDir\winsweep.wixobj"
if ($LASTEXITCODE -ne 0) { throw "light.exe falhou (exit code $LASTEXITCODE)" }

Write-Host "==> Instalador gerado em: $outputMsi" -ForegroundColor Green
