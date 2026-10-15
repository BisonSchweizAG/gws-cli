<div align="center">
  <img src="icon/icon.svg" alt="GWS Icon" width="100"/>
</div>

# gws - Google Cloud Workstation CLI

A command-line tool to manage Google Cloud Workstations.

## Description

`gws` is a utility to simplify the management of Google Cloud Workstations. It provides commands to start, stop, and
connect to your workstation, as well as manage your configuration.

## Installation

You can install `gws` by downloading a binary from
the [latest release](https://github.com/BisonSchweizAG/gws-cli/releases/latest) or via Go using `go install`:

```bash
go install github.com/bisonschweizag/gws-cli@latest
```

## Usage

### Global Flags

Every command supports these global flags:

| Flag       | Description                                                           |
|------------|-----------------------------------------------------------------------|
| `--config` | Path to the configuration file (default: `~/.config/gws/config.yaml`) |
| `--ctx`    | The context to use instead of the current one                         |

### Commands

#### `gws setup`

Create or update `config.yaml` by running an interactive setup wizard.

```bash
gws setup
```

#### `gws start [context]`

Start the workstation for the given or current context.

```bash
gws start
gws start my-workstation
```

#### `gws stop [context]`

Stop the workstation for the given or current context.

```bash
gws stop
gws stop my-workstation
gws stop --all
```

| Flag    | Description            |
|---------|------------------------|
| `--all` | Stop all workstations  |

#### `gws restart [context]`

Restart the workstation for the given or current context.

```bash
gws restart
gws restart my-workstation
```

#### `gws status`

List all configured workstations with their current state. Alias: `st`.

```bash
gws status
gws status -o wide
gws status -o transpose
```

| Flag           | Description                          |
|----------------|--------------------------------------|
| `-o, --output` | Output format: `wide` or `transpose` |

#### `gws list`

List all cloud workstations in the current context. Aliases: `ls`, `ps`.

```bash
gws list
gws list -o wide
gws list -s name
gws list -f john
gws list -r
gws ps
```

| Flag            | Description                                                                                            |
|-----------------|--------------------------------------------------------------------------------------------------------|
| `-o, --output`  | Output format: `wide`                                                                                  |
| `-s, --sort`    | Sort by column: `name`, `user`, `project`, `config`, `state`, `last-started` (default: `last-started`) |
| `-f, --filter`  | Filter workstations by name or user (case-insensitive)                                                 |
| `-r, --running` | Show only running workstations (automatic when using `ps` alias)                                       |

#### `gws tunnel [context]`

Create an SSH tunnel to the workstation.

```bash
gws tunnel
gws tunnel my-workstation
gws tunnel -p 2222
gws tunnel --restart
```

| Flag                    | Description                                                       |
|-------------------------|-------------------------------------------------------------------|
| `-p, --local-host-port` | The local host port to open (default is the port from the config) |
| `--check-token`         | Enable periodic token check (default: `true`)                     |
| `--restart`             | Restart the workstation before opening the tunnel                 |

#### `gws launch [context]`

Open the default web URL of the cloud workstation in your browser.

```bash
gws launch
gws launch my-workstation
```

#### `gws up`

Upload files and directories to the workstation as defined in the context configuration.

```bash
gws up
gws up my-workstation
```

#### `gws down`

Download files and directories from the workstation as defined in the context configuration.

```bash
gws down
gws down my-workstation
```

#### `gws ctx [context]`

Switch the current context. If no context is provided, an interactive selection is shown.

```bash
gws ctx
gws ctx my-workstation
gws ctx --current
```

| Flag        | Description                       |
|-------------|-----------------------------------|
| `--current` | Print the current active context  |

#### `gws scripts win-reconnect-ssh [context]`

Generate a Windows SSH reconnect script.

```bash
gws scripts win-reconnect-ssh
gws scripts win-reconnect-ssh -s
gws scripts win-reconnect-ssh -o ./reconnect.bat
```

| Flag           | Description                                    |
|----------------|------------------------------------------------|
| `-o, --output` | Output file (default: stdout)                  |
| `-s, --save`   | Save the script to the default config location |

#### `gws scripts bash-reconnect-ssh [context]`

Generate a Bash SSH reconnect script.

```bash
gws scripts bash-reconnect-ssh
gws scripts bash-reconnect-ssh -s
gws scripts bash-reconnect-ssh -o ./reconnect.sh
```

| Flag           | Description                                    |
|----------------|------------------------------------------------|
| `-o, --output` | Output file (default: stdout)                  |
| `-s, --save`   | Save the script to the default config location |

#### `gws logout`

Log out and remove the stored Google access token.

```bash
gws logout
```

#### `gws update`

Update `gws` to the latest release from GitHub.

```bash
gws update
gws update --check
gws update --force
```

| Flag      | Description                                                                 |
|-----------|-----------------------------------------------------------------------------|
| `--check` | Only check whether a newer version is available                             |
| `--force` | Install the latest release even if it is not newer (or this is a dev build) |

## Configuration

`gws` is configured using a YAML file (default: `~/.config/gws/config.yaml`). You can use the `gws setup` command to
create an initial configuration.

The configuration file can contain multiple contexts. Each context defines the connection details for a specific
workstation.

### `config.yaml` example

```yaml
currentContext: my-workstation
contexts:
  my-workstation:
    host: localhost
    port: 2222
    user: user
    privateKeyFile: /path/to/your/private/key
    knownHostsFile: /path/to/your/known_hosts
    postConnectCommand: # open 'GWS' profile in a new Windows Terminal Tab
      - 'wt'
      - '--window=0'
      - '--profile=GWS'
    gcloud:
      project: my-project
      account: user@example.com
      region: a-region
      cluster: my-cluster
      config: my-workstation-config
      name: my-workstation
    dirs:
      - path: /home/user/.ssh
        permissions: "0700"
    files:
      - sourcePath: /path/to/your/file
        path: /home/user/file
        permissions: "0644"
        direction: up
chromeBrowser:
  executablePath: C:\Program Files\Google\Chrome\Application\chrome.exe
  profileDirectory: Profile X
jetbrainsGateway:
  downloadDestination: C:\workspace\Jetbrains-Programs\JetBrainsClientDist
```

### Configuration Options

- `currentContext`: The name of the currently active context.
- `contexts`: A map of contexts.
    - `<context-name>`:
        - `host`: The hostname or IP address of the workstation.
        - `port`: The port to connect to.
        - `user`: The username to use for the SSH connection.
        - `privateKeyFile`: The path to the private key for the SSH connection.
        - `knownHostsFile`: The path to the known hosts file for the SSH connection.
        - `postConnectCommand`: Optional command and arguments to start after the tunnel is opened. The first item is
          the executable, and the remaining items are passed as arguments.
        - `gcloud`: The Google Cloud configuration.
            - `project`: The Google Cloud project.
            - `account`: The Google Cloud account (email, optional).
            - `region`: The Google Cloud region.
            - `cluster`: The Google Cloud cluster.
            - `config`: The workstation configuration.
            - `name`: The name of the workstation.
        - `dirs`: A list of directories to create on the workstation.
            - `path`: The path of the directory.
            - `permissions`: The permissions of the directory.
        - `files`: A list of files to upload to the workstation.
            - `sourcePath`: The path of the local file.
            - `path`: The path of the remote file.
            - `permissions`: The permissions of the remote file.
            - `direction`: The direction (up / down) the file is copied.
- `sshTimeoutSeconds`: Optional SSH connection timeout in seconds (default: `30`).
- `startTimeoutSeconds`: Optional workstation start timeout in seconds (default: `300`).
- `chromeBrowser`: Optional Chrome browser configuration.
    - `executablePath`: The path to the Chrome executable.
    - `profileDirectory`: The Chrome profile directory to use. (check profile dir by opening `chrome://version` in
      chrome)
- `jetbrainsGateway`: Optional JetBrains Gateway configuration.
    - `downloadDestination`: The directory where JetBrains programs are downloaded.

## Setup in JetBrains Gateway

1. Create a new SSH Connection  
   ![Gateway Homescreen](docs/gateway-ssh-new-home.png "Gateway Homescreen")

2. Configure the SSH Connection

- Username: `user`
- Host: `localhost`
- Port: port configured for your gws context

![Gateway SSH Connection](docs/gateway-ssh-new-connection.png "Gateway SSH Connection")

## Integrate into Windows Terminal

1. Generate an SSH Start Script for Windows

The generated script automatically reconnects to the Workstation if the tunnel is re-opened.

```text
# dry run
gws scripts win-reconnect-ssh

# save the file to %HOME%\.config\gws\win-reconnect-ssh.bat
gws scripts win-reconnect-ssh -s
```

This saves a cmd file in `%HOME%\.config\gws\`.

2. Create a new Profile in Windows Terminal, giving it a name and select the cmd file as executable.
   ![New Windows Terminal Profile](docs/windows-terminal-profile.png "New Windows Terminal Profile")

3. Configure the new profile in the postConnectCommand of your context in `%HOME%\.config\gws\config.yaml`.

```yaml
contexts:
  my-workstation:
    postConnectCommand: # open 'GWS PROD' profile in a new Windows Terminal Tab
      - 'wt'
      - '--window=0'
      - '--profile=GWS PROD'
```

## Windows SmartScreen

On some systems, Windows SmartScreen may block the execution of the gws.exe. To allow the gws to run, verify you have
downloaded it from this repository and verify the file is signed by gws certificate `gws-cli.bison-group.com`.

![SmartScreen Lock](docs/windows-smart-screen.png "SmartScreen Lock")

If this is the case, right-click on the gws.exe file and select `Properties` / `Eigenschaften`.

In the `Properties` / `Eigenschaften` dialog, click on the `Unblock` / `Zulassen` button.

![Unblock](docs/windows-properties.png "Unblock")

## MacOS Binary Unlock

On some systems, MacOS may block the execution of the gws binary. To allow the gws to run, verify you have
downloaded it from this repository. Then you open a terminal and execute the following command, where ```gws``` is the full path to gws binary :

```sudo xattr -d com.apple.quarantine gws```

## Android / Termux

You can run `gws` on Android using [Termux](https://termux.com/), a terminal emulator for Android.

### Installation

1. Install Termux from [F-Droid](https://f-droid.org/packages/com.termux/) (recommended) or the Google Play Store.
2. Download the Android release binary from the [latest release](https://github.com/BisonSchweizAG/gws-cli/releases/latest)
   (look for the `android_arm64` archive).
3. Extract the binary and make it executable:

```bash
tar xzf gws_*_android_arm64.tar.gz
chmod +x gws
```

4. Optionally, move the binary to a directory in your `$PATH`:

```bash
mv gws $HOME/.local/bin/
```

### Building from Source

To build the Android binary from source, set the Google OAuth client credentials as environment variables:

```bash
export GOOGLE_OIDC_CLIENT_ID="your-client-id"
export GOOGLE_OIDC_CLIENT_SECRET="your-client-secret"
make build-android
```

Without these variables, the binary will not be able to authenticate with Google Cloud.

### Authentication

Use the `gws login` command to authenticate. Since the Android browser cannot reach a local HTTP server in Termux,
the login flow uses a manual authorization code:

1. Run `gws login` — it will print an authentication URL.
2. Open the URL in your device's browser.
3. Sign in and grant permission.
4. After authentication, the browser will show a URL containing an authorization code — copy the `code` parameter value.
5. Paste the code back into the Termux session when prompted.

### DNS Resolution

If you encounter DNS resolution issues (e.g., `dial tcp: lookup ...: no such host`), set the `GODEBUG` environment
variable before running `gws`:

```bash
export GODEBUG=netdns=go
gws status
```

This forces Go to use its pure-Go DNS resolver, which works better on Android's Bionic libc. You can add this to
your `~/.bashrc` or `~/.zshrc` in Termux to make it permanent.

### Configuration

The configuration file is stored at `~/.config/gws/config.yaml` (same as other platforms). Use `gws setup` to create
an initial configuration interactively.

### Known Issues

On some versions of Termux (Android 15+), there's a known bug where Go programs receive their own path as an extra
command-line argument. This causes an error like `unknown command "/data/data/com.termux/files/home/gws"`. The fix is
included in recent builds — if you encounter this issue, try downloading the latest release binary.