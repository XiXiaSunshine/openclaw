@ECHO off
GOTO start
:find_dp0
SET dp0=%~dp0
EXIT /b
:start
SETLOCAL
CALL :find_dp0
"%dp0%\node_modules\oh-my-opencode-windows-x64\bin\oh-my-opencode.exe"   %*
