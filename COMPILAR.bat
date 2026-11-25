@echo off
cd /d "%~dp0"
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags="-H=windowsgui -s -w" -o "osu-Auto-Direct.exe" .
if errorlevel 1 exit /b 1
echo Compilacion completada.
