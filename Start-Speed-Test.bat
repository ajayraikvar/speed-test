@rem Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See LICENSE and NOTICE.
@echo off
setlocal EnableExtensions
cd /d "%~dp0"
title Speed Test - by Ajay Raikvar

echo.
echo ============================================================
echo   SPEED TEST - SELF-HOSTED NETWORK PERFORMANCE ANALYZER
echo   Developed by www.ajayraikvar.com
echo ============================================================
echo.
echo Other PCs only need a modern browser. No client installation.
echo Online: checks for the latest official release before starting.
echo Offline: uses the bundled or previously installed server.
echo.

set "HOST_PORT=80"
set /p "HOST_PORT=Enter Host Port [80]: "
if not defined HOST_PORT set "HOST_PORT=80"
rem PowerShell performs strict numeric port validation before starting the server.

if not exist "%~dp0Start-Speed-Test.ps1" (
  echo ERROR: Start-Speed-Test.ps1 is missing.
  echo Keep the BAT and PowerShell launcher together, or extract the full offline ZIP.
  pause
  exit /b 3
)

powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start-Speed-Test.ps1" -Port "%HOST_PORT%"
set "EXIT_CODE=%ERRORLEVEL%"
echo.
if not "%EXIT_CODE%"=="0" echo Speed Test exited with code %EXIT_CODE%.
pause
exit /b %EXIT_CODE%
