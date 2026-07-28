@echo off
echo Building Midir Overlay (Wails v2)...
cd overlay
wails build
echo Done! Binary created at overlay\build\bin\overlay.exe
pause
