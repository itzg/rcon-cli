[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/itzg/rcon-cli)](https://github.com/itzg/rcon-cli/releases/latest)
[![CircleCI](https://img.shields.io/circleci/build/github/itzg/rcon-cli)](https://app.circleci.com/pipelines/github/itzg/rcon-cli)


A little RCON cli based on james4k's RCON library for golang.

## Installation

1. Download the appropriate binary for your platform from the [latest releases](https://github.com/itzg/rcon-cli/releases/latest)

2. On UNIX-y platforms, set the binary to be executable

Done.

## Usage

```text
rcon-cli is a CLI for attaching to an RCON enabled game server, such as Minecraft.
Without any additional arguments, the CLI will start an interactive session with
the RCON server.

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
      --config string     config file (default is $HOME/.rcon-cli.yaml)
      --host string       RCON server's hostname (default "localhost")
      --password string   RCON server's password
      --port int          Server's RCON port (default 27015)
```

## Configuration

You can preconfigure rcon-cli to use the arguments you want by default by modifying the file `.rcon-cli.yaml` in your home folder. If you want to use any other file use the argument `--config /path/to/the/config.yaml`. 

Example of a `.rcon-cli.yaml` file:
```yaml
host: mydomain.com
port: 12345
password: mycustompassword
```

That way executing `rcon-cli` without arguments would connect to `mydomain.com:12345` with the password `mycustompassword` by default.
