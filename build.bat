@echo off
setlocal EnableExtensions
pushd "%~dp0"
if errorlevel 1 (
  echo ERROR: Could not enter Midir source directory.
  exit /b 1
)

if not exist "front\package.json" (
  echo ERROR: Missing front\package.json. Extract the full source archive before building.
  goto :fail
)

echo [1/5] Installing frontend dependencies...
pushd front
if errorlevel 1 goto :fail
call npm ci
if errorlevel 1 (
  popd
  echo ERROR: Frontend dependency installation failed.
  goto :fail
)

echo [2/5] Building frontend...
call npm run build
if errorlevel 1 (
  popd
  echo ERROR: Frontend build failed. See the npm errors above.
  goto :fail
)
popd

if not exist "front\dist\index.html" (
  echo ERROR: Frontend build did not produce front\dist\index.html.
  goto :fail
)

echo [3/5] Moving static files to backend...
if exist "cmd\dilmeterapi\static" rmdir /s /q "cmd\dilmeterapi\static"
if exist "cmd\dilmeterapi\static" (
  echo ERROR: Could not remove old static directory.
  goto :fail
)
mkdir "cmd\dilmeterapi\static"
if errorlevel 1 goto :fail
xcopy "front\dist\*" "cmd\dilmeterapi\static\" /E /I /Y >nul
if errorlevel 1 (
  echo ERROR: Copying compiled frontend assets failed.
  goto :fail
)
if not exist "cmd\dilmeterapi\static\index.html" (
  echo ERROR: Compiled frontend index.html missing from embedded static directory.
  goto :fail
)

echo [4/5] Tidying Go modules...
go mod tidy
if errorlevel 1 (
  echo ERROR: Go module tidy failed.
  goto :fail
)

echo [5/5] Building Go executable...
if not exist "build" mkdir "build"
if errorlevel 1 goto :fail
if exist "build\Midir.exe" del /q "build\Midir.exe"
go build -ldflags="-s -w" -trimpath -v -o "build\Midir.exe" ./cmd/dilmeterapi
if errorlevel 1 (
  echo ERROR: Go build failed. See the Go error above.
  goto :fail
)
if not exist "build\Midir.exe" (
  echo ERROR: Go returned without a Midir.exe output.
  goto :fail
)

echo.
echo Build Complete! Executable is in the 'build' folder.
popd
endlocal
pause
exit /b 0

:fail
echo.
echo BUILD FAILED. No new executable was produced. Read the first error above.
popd
endlocal
pause
exit /b 1
