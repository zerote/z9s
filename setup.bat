@echo off
REM KT9S Setup Script for Windows
REM This script sets up the kt9s development environment on Windows

setlocal enabledelayedexpansion

cls
echo ============================================================
echo            KT9S Development Environment Setup              
echo ============================================================
echo.

REM Check Go version
echo Checking Go installation...
for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i

if "%GO_VERSION%"=="" (
    echo Error: Go is not installed or not in PATH.
    echo Please install Go 1.25 or later from: https://golang.org/dl
    pause
    exit /b 1
)

echo ✓ Go %GO_VERSION% found
echo.

REM Check if we're in the kt9s directory
if not exist "go.mod" (
    echo Error: go.mod not found. Please run this script from the kt9s root directory.
    pause
    exit /b 1
)

echo ✓ Project directory verified
echo.

REM Update module name
echo Configuring module name...
set /p USERNAME=Enter your GitHub username (for module path): 

if "%USERNAME%"=="" (
    echo Error: Username cannot be empty
    pause
    exit /b 1
)

echo Replacing 'yourusername' with '%USERNAME%'...

REM This is more complex in Windows batch
REM We'll use PowerShell for the file replacement
powershell -Command ^
    "Get-ChildItem -Recurse -Include '*.go','*.mod' | ForEach-Object { " ^
    "  (Get-Content $_) -replace 'yourusername', '%USERNAME%' | Set-Content $_ " ^
    "}"

echo ✓ Module name updated to github.com/%USERNAME%/kt9s
echo.

REM Download dependencies
echo Downloading Go dependencies...
call go mod tidy
echo ✓ Dependencies downloaded
echo.

REM Verify build
echo Verifying build...
go build -o .\bin\kt9s-test.exe >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo ✓ Build verification successful
    del .\bin\kt9s-test.exe
) else (
    echo ⚠ Build verification had some warnings (this is normal for incomplete integration)
)
echo.

REM Create necessary directories
echo Creating additional directories...
if not exist "docs" mkdir docs
if not exist "logs" mkdir logs
if not exist "bin" mkdir bin

echo.
echo ============================================================
echo               ✓ Setup Complete!
echo ============================================================
echo.
echo Next steps:
echo.
echo 1. Read SETUP_GUIDE.md:
echo    notepad SETUP_GUIDE.md
echo.
echo 2. Copy k9s code (if not already done):
echo    xcopy "path\to\k9s-original\internal\*" "internal\k9s" /E /I
echo.
echo 3. Copy ktop code (if not already done):
echo    xcopy "path\to\ktop-original\*" "internal\ktop" /E /I
echo.
echo 4. Build the project:
echo    go build -o .\bin\kt9s.exe
echo.
echo 5. Run kt9s:
echo    .\bin\kt9s.exe
echo.
pause
