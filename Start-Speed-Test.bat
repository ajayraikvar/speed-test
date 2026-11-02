@rem Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See LICENSE and NOTICE.
@echo off
setlocal EnableExtensions
cd /d "%~dp0"
title Speed Test v2.0 - by Ajay Raikvar

echo.
echo ============================================================
echo   SPEED TEST v2.0 - NETWORK PERFORMANCE ANALYZER
echo   Developed by www.ajayraikvar.com
echo ============================================================
echo.
echo Host PC runs the server. Other PCs need only a modern browser.
echo Online: checks for the latest GitHub release and verifies SHA-256.
echo Offline: starts this bundled version or a previously cached version.
echo.

set "HOST_PORT=80"
set /p "HOST_PORT=Enter Host Port [80]: "
if not defined HOST_PORT set "HOST_PORT=80"

set "MAX_CLIENTS=16"
set /p "MAX_CLIENTS=Maximum browser clients [16]: "
if not defined MAX_CLIENTS set "MAX_CLIENTS=16"

if not exist "%~dp0Start-Speed-Test.ps1" (
  echo ERROR: Start-Speed-Test.ps1 is missing. Extract the complete package.
  pause
  exit /b 3
)
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start-Speed-Test.ps1" -Port "%HOST_PORT%" -MaxClients "%MAX_CLIENTS%"
set "EXIT_CODE=%ERRORLEVEL%"
echo.
if not "%EXIT_CODE%"=="0" echo Speed Test exited with code %EXIT_CODE%.
pause
exit /b %EXIT_CODE%
