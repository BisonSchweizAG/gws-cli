#!/bin/bash
# {{.FileName}}

# This script automatically reconnects to an SSH server when the connection drops.
# It continuously attempts to establish an SSH connection and retries after a specified delay if disconnected.

# The number of seconds to wait before retrying the connection.
RETRY_SECONDS=3

# Switch off terminal mouse tracking (modes 1000/1002/1003/1006/1015).
# A dropped connection can leave the local terminal in mouse-reporting mode, which prints raw
# sequences like 35;61;11M on mouse move and breaks text selection/copy.
# Disabling modes that were never enabled is harmless.
reset_mouse_tracking() {
    printf '\033[?1000l\033[?1002l\033[?1003l\033[?1006l\033[?1015l'
}

while true; do
    ssh {{.User}}@{{.Host}} -p {{.Port}}{{ if .KnownHostsFile }} -o UserKnownHostsFile={{.KnownHostsFile}}{{ end }}
    ssh_rc=$?
    reset_mouse_tracking
    if [ $ssh_rc -eq 0 ]; then
        break
    fi
    clear
    echo "Disconnected. Reconnecting in $RETRY_SECONDS seconds..."
    sleep $RETRY_SECONDS
    clear
done
