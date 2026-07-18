@echo off
chcp 65001 >nul
cd /d "%~dp01szt"

echo ================================
echo  编译 net_work
echo ================================

echo.
echo [1/2] 编译 Linux amd64 ...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o bin/net_work-linux-amd64 ./main
if %errorlevel% equ 0 ( echo   ✓ 完成 ) else ( echo   ✗ 失败 & pause & exit /b 1 )

echo.
echo [2/2] 编译 Windows amd64 ...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o bin/net_work-windows-amd64.exe ./main
if %errorlevel% equ 0 ( echo   ✓ 完成 ) else ( echo   ✗ 失败 & pause & exit /b 1 )

echo.
echo ================================
echo  全部编译完成！
echo  输出目录: bin/
echo ================================
pause
