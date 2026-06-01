#requires -Version 5.1
# Compila conversations-viewer.exe (equivalente Windows del script 'build').
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

Write-Host "Compilando conversations-viewer.exe..."
& go build -o conversations-viewer.exe .
if ($LASTEXITCODE -ne 0) { throw "go build fallo con codigo $LASTEXITCODE" }
Write-Host "Compilacion exitosa: $PSScriptRoot\conversations-viewer.exe"
