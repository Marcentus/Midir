@echo off
pushd %~dp0

echo [1/5] Installing frontend dependencies...
cd front
call npm install

echo [2/5] Building frontend...
call npm run build
cd ..

echo [3/5] Moving static files to backend...
if exist "cmd\dilmeterapi\static" rmdir /s /q "cmd\dilmeterapi\static"
mkdir "cmd\dilmeterapi\static"
type nul > "cmd\dilmeterapi\static\.keep"
xcopy front\dist cmd\dilmeterapi\static /E /I /Y

echo [4/5] Tidying Go modules...
go mod tidy

echo [5/5] Building Go executable...
go build -ldflags="-s -w" -trimpath -v -o build/Midir.exe ./cmd/dilmeterapi

echo.
echo Build Complete! Executable is in the 'build' folder.
popd
pause
