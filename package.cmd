@echo off
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0package.ps1" %*
exit /b %errorlevel%
