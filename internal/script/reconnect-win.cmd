REM {{.FileName}}
@echo off
setlocal

REM This script automatically reconnects to an SSH server when the connection drops.
REM It continuously attempts to establish an SSH connection and retries after a specified delay if disconnected.

REM The number of seconds to wait before retrying the connection.
set "RETRY_SECONDS=3"

REM Build an ESC character (used to reset the terminal's mouse-tracking modes).
for /F %%a in ('echo prompt $E ^| cmd') do set "ESC=%%a"

:reconnect_loop
ssh {{.User}}@{{.Host}} -p {{.Port}}{{ if .KnownHostsFile }} -o UserKnownHostsFile={{.KnownHostsFile}}{{ end }}
set "SSH_RC=%errorlevel%"

REM After the session ends (exit, drop, Ctrl+C), switch off mouse tracking (modes 1000/1002/1003/1006/1015).
REM A dropped connection can leave the local terminal in mouse-reporting mode, which prints raw
REM sequences like 35;61;11M on mouse move and breaks text selection/copy.
REM Disabling modes that were never enabled is harmless.
<nul set /p "=%ESC%[?1000l%ESC%[?1002l%ESC%[?1003l%ESC%[?1006l%ESC%[?1015l"

if %SSH_RC% == 0 goto end
cls
echo Disconnected. Reconnecting in %RETRY_SECONDS% seconds...
timeout /t %RETRY_SECONDS% /nobreak
cls
goto reconnect_loop

:end
