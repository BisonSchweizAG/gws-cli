REM {{.FileName}}
@echo off
setlocal

REM This script automatically reconnects to an SSH server when the connection drops.
REM It continuously attempts to establish an SSH connection and retries after a specified delay if disconnected.

REM The number of seconds to wait before retrying the connection.
set "RETRY_SECONDS=3"

:reconnect_loop
ssh {{.User}}@{{.Host}} -p {{.Port}}{{ if .KnownHostsFile }} -o UserKnownHostsFile={{.KnownHostsFile}}{{ end }}
if %errorlevel% == 0 goto end
cls
echo Disconnected. Reconnecting in %RETRY_SECONDS% seconds...
timeout /t %RETRY_SECONDS% /nobreak
cls
goto reconnect_loop

:end