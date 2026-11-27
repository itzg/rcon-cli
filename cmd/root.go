// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/itzg/go-flagsfiller"
	"github.com/itzg/rcon-cli/cli"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Config holds every rcon-cli setting. Flags, environment variables and
// the config file all feed into this one struct.
type Config struct {
	Host         string `default:"localhost" usage:"RCON server's hostname"`
	Password     string `usage:"RCON server's password"`
	Port         int    `default:"25575" usage:"Server's RCON port"`
	AbortOnCtrlC bool   `default:"true" usage:"Abort on ctrl-c rather than clearing line"`
}

var (
	cfgFile string
	config  Config
)

// fileConfig mirrors Config but with pointer fields, so a config file
// that omits a key can be told apart from one that sets it. A missing
// key must not overwrite the flag, env or default value.
type fileConfig struct {
	Host         *string `yaml:"host"`
	Password     *string `yaml:"password"`
	Port         *int    `yaml:"port"`
	AbortOnCtrlC *bool   `yaml:"abort-on-ctrl-c"`
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "rcon-cli [flags] [RCON command ...]",
	Short: "A CLI for attaching to an RCON enabled game server",
	Example: `
rcon-cli --host mc1 --port 25575
rcon-cli --port 25575 stop
RCON_PORT=25575 rcon-cli stop
`,
	Long: `
rcon-cli is a CLI for attaching to an RCON enabled game server, such as Minecraft.
Without any additional arguments, the CLI will start an interactive session with
the RCON server. Send the keyword "exit" to close the session. The CLI will exit.

If arguments are passed into the CLI, then the arguments are sent
as a single command (joined by spaces), the response is displayed,
and the CLI will exit.
`,
	Run: func(cmd *cobra.Command, args []string) {
		hostPort := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

		if len(args) == 0 {
			cli.Start(hostPort, config.Password, os.Stdout, config.AbortOnCtrlC)
		} else {
			cli.Execute(hostPort, config.Password, os.Stdout, args...)
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/rcon-cli/config.yaml or $HOME/.rcon-cli.yaml)")

	filler := flagsfiller.New(flagsfiller.WithEnv("RCON"))
	var fs flag.FlagSet
	if err := filler.Fill(&fs, &config); err != nil {
		log.Fatal(err)
	}
	RootCmd.PersistentFlags().AddGoFlagSet(&fs)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	path := cfgFile
	if path == "" {
		path = findUserConfigFile()
	}
	if path == "" {
		for _, dir := range []string{"$HOME", "/data", "/server"} {
			if p := findConfigFileIn(dir); p != "" {
				path = p
				break
			}
		}
	}
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("reading config file %s: %v", path, err)
	}
	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		log.Fatalf("parsing config file %s: %v", path, err)
	}
	applyConfigFile(&fc)
}

// applyConfigFile copies a config file value into config only where the
// user did not set the value by flag or by env, so the precedence stays
// flags > env > config file > defaults.
func applyConfigFile(fc *fileConfig) {
	flags := RootCmd.PersistentFlags()
	if fc.Host != nil && !flags.Changed("host") && os.Getenv("RCON_HOST") == "" {
		config.Host = *fc.Host
	}
	if fc.Password != nil && !flags.Changed("password") && os.Getenv("RCON_PASSWORD") == "" {
		config.Password = *fc.Password
	}
	if fc.Port != nil && !flags.Changed("port") && os.Getenv("RCON_PORT") == "" {
		config.Port = *fc.Port
	}
	if fc.AbortOnCtrlC != nil && !flags.Changed("abort-on-ctrl-c") && os.Getenv("RCON_ABORT_ON_CTRL_C") == "" {
		config.AbortOnCtrlC = *fc.AbortOnCtrlC
	}
}

// findUserConfigFile looks for a config file in the user's config directory, such as
// ~/.config/rcon-cli/config.yaml on Linux. It returns an empty string if there is none.
func findUserConfigFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return findConfigFileIn(filepath.Join(dir, "rcon-cli"))
}

// configFileExtensions are the config file formats rcon-cli reads.
// YAML is the only format the README documents.
var configFileExtensions = []string{"yaml", "yml"}

// findConfigFileIn returns the path of the first config.<ext> file in dir,
// or an empty string if there is none.
func findConfigFileIn(dir string) string {
	for _, ext := range configFileExtensions {
		candidate := filepath.Join(dir, "config."+ext)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}
