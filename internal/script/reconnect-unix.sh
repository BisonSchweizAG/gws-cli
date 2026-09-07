#!/bin/bash
# {{.FileName}}

# This script automatically reconnects to an SSH server when the connection drops.
# It continuously attempts to establish an SSH connection and retries after a specified delay if disconnected.

# The number of seconds to wait before retrying the connection.
RETRY_SECONDS=3

while true; do
    ssh {{.User}}@{{.Host}} -p {{.Port}}{{ if .KnownHostsFile }} -o UserKnownHostsFile={{.KnownHostsFile}}{{ end }}
    if [ $? -eq 0 ]; then
        break
    fi
    clear
    echo "Disconnected. Reconnecting in $RETRY_SECONDS seconds..."
    sleep $RETRY_SECONDS
    clear
done