[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/itzg/rcon-cli)](https://github.com/itzg/rcon-cli/releases/latest)
[![test](https://github.com/itzg/rcon-cli/actions/workflows/test.yml/badge.svg)](https://github.com/itzg/rcon-cli/actions/workflows/test.yml)


A little RCON cli for Minecraft and other game servers that speak the Source RCON protocol.

## Installation

1. Download the appropriate binary for your platform from the [latest releases](https://github.com/itzg/rcon-cli/releases/latest)

2. On UNIX-y platforms, set the binary to be executable - done!

If you [have Go](https://golang.org/dl/) you could also simply `go install github.com/itzg/rcon-cli@latest && rcon-cli -h`.

## Usage

```text
rcon-cli is a CLI for attaching to an RCON enabled game server, such as Minecraft.
Without any additional arguments, the CLI will start an interactive session with
the RCON server. Send the keyword "exit" to close the session. The CLI will exit.

If arguments are passed into the CLI, then the arguments are sent
as a single command (joined by spaces), the response is displayed,
and the CLI will exit.

Usage:
  rcon-cli [flags] [RCON command ...]

Examples:

rcon-cli --host mc1 --port 25575
rcon-cli --port 25575 stop
RCON_PORT=25575 rcon-cli stop


Flags:
      --config string     config file (default is $XDG_CONFIG_HOME/rcon-cli/config.yaml or $HOME/.rcon-cli.yaml)
      --host string       RCON server's hostname (default "localhost")
      --password string   RCON server's password
      --port int          Server's RCON port (default 25575)
      --abort-on-ctrl-c=true|false    If Ctrl-c should abort or just clear line, defaults to true (abort)
```

## Configuration

You can preconfigure rcon-cli to use the arguments you want by default with a config file. The config file is YAML (`.yaml` or `.yml`). rcon-cli uses the first one it finds of:

1. the file given with `--config /path/to/the/config.yaml`
2. `rcon-cli/config.yaml` in your user config directory: `$XDG_CONFIG_HOME` or `~/.config` on Linux, `~/Library/Application Support` on macOS, `%AppData%` on Windows
3. `.rcon-cli.yaml` in your home folder, then in `/data` and `/server`

Example of a `.rcon-cli.yaml` file:
```yaml
host: mydomain.com
port: 12345
password: mycustompassword
```

That way executing `rcon-cli` without arguments would connect to `mydomain.com:12345` with the password `mycustompassword` by default.
