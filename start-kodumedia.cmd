@echo off
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0start-kodumedia.ps1"
exit /b %errorlevel%
