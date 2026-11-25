@echo off
if exist "%LOCALAPPDATA%\OsuAutoDirect\osu-Auto-Direct.exe" (
  start "" "%LOCALAPPDATA%\OsuAutoDirect\osu-Auto-Direct.exe" --uninstall
) else (
  echo No se encontro la instalacion.
  pause
)
