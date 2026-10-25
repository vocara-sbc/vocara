@echo off
chcp 936 >nul
setlocal enabledelayedexpansion

rem Vocara 统一入口（Windows）：实际逻辑在 run.sh，本脚本负责定位 bash 并转发。
set "ROOT=%~dp0"
set "BASH_EXE="

rem 1. PATH 中存在 bash 时直接使用
where bash >nul 2>nul
if not errorlevel 1 (
  set "BASH_EXE=bash"
  goto :run
)

rem 2. 常见安装目录
for %%P in (
  "%ProgramFiles%\Git\bin\bash.exe"
  "%ProgramFiles(x86)%\Git\bin\bash.exe"
  "%LOCALAPPDATA%\Programs\Git\bin\bash.exe"
) do (
  if exist %%~P (
    set "BASH_EXE=%%~P"
    goto :run
  )
)

rem 3. 注册表中的 Git 安装路径
for %%K in (
  "HKLM\SOFTWARE\GitForWindows"
  "HKLM\SOFTWARE\WOW6432Node\GitForWindows"
  "HKCU\SOFTWARE\GitForWindows"
) do (
  for /f "tokens=2,*" %%A in ('reg query %%~K /v InstallPath 2^>nul') do (
    if exist "%%B\bin\bash.exe" (
      set "BASH_EXE=%%B\bin\bash.exe"
      goto :run
    )
  )
)

echo [ERROR] 未检测到 bash，请先安装 Git for Windows：https://git-scm.com/download/win
exit /b 1

:run
"!BASH_EXE!" "%ROOT%run.sh" %*
exit /b %errorlevel%
