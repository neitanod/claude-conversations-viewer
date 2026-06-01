#requires -Version 5.1
# Ejecuta los tests (equivalente Windows del script 'test').
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

& go test -v -cover @args
exit $LASTEXITCODE
