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
	"fmt"
	"log"
	"os"

	"github.com/itzg/rcon-cli/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"strconv"
)

var (
	cfgFile string
)

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
the RCON server.

If arguments are passed into the CLI, then the arguments are sent
as a single command (joined by spaces), the response is displayed,
and the CLI will exit.
`,
	Run: func(cmd *cobra.Command, args []string) {

		hostPort := net.JoinHostPort(viper.GetString("host"), strconv.Itoa(viper.GetInt("port")))
		password := viper.GetString("password")

		if len(args) == 0 {
			cli.Start(hostPort, password, os.Stdin, os.Stdout)
		} else {
			cli.Execute(hostPort, password, os.Stdout, args...)
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

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rcon-cli.yaml)")
	RootCmd.PersistentFlags().String("host", "localhost", "RCON server's hostname")
	RootCmd.PersistentFlags().String("password", "", "RCON server's password")
	RootCmd.PersistentFlags().Int("port", 25575, "Server's RCON port")
	err := viper.BindPFlags(RootCmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(".rcon-cli") // name of config file (without extension)
		viper.AddConfigPath("$HOME")     // adding home directory as first search path
	}

	// This will allow for env vars like RCON_PORT
	viper.SetEnvPrefix("rcon")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
