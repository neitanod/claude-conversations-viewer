#requires -Version 5.1
# Ejecuta conversations-viewer.exe (equivalente Windows del script 'run').
# Si el binario no existe, lo compila primero.
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

if (-not (Test-Path .\conversations-viewer.exe)) {
    Write-Host "Binario no encontrado, compilando..."
    & "$PSScriptRoot\build.ps1"
}

& .\conversations-viewer.exe @args
exit $LASTEXITCODE
