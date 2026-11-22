@echo off
rem Week3_C2 demo: run fixed then buggy, pause between for screenshots
rem go is the portable copy under .tools, no system install needed
cd /d %~dp0
set GOEXE=C:\Users\admin\WorkBuddy\2026-11-22-17-03-03\.tools\go\bin\go.exe

echo ============================================
echo  1/2  fixed  version (should exit cleanly)
echo ============================================
"%GOEXE%" run repro_k8s_mux.go -version=fixed
echo.
echo [exit code above should be 0 / no fatal]
pause

echo.
echo ============================================
echo  2/2  buggy  version (fatal deadlock expected!)
echo ============================================
"%GOEXE%" run repro_k8s_mux.go -version=buggy
echo.
echo [fatal error: all goroutines are asleep - deadlock!  -- this is EXPECTED]
pause
