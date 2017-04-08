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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"github.com/itzg/rcon-cli/cli"
	"strconv"
)

var (
	cfgFile string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "rcon-cli",
	Short: "A CLI for attaching to an RCON enabled game server",
	Long:  ``,
	Run:   func(cmd *cobra.Command, args []string) {
		hostPort := net.JoinHostPort(viper.GetString("host"), strconv.Itoa(viper.GetInt("port")))
		cli.Start(hostPort, viper.GetString("password"), os.Stdin, os.Stdout)
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
	RootCmd.PersistentFlags().Int("port", 27015, "Server's RCON port")
	viper.BindPFlags(RootCmd.PersistentFlags())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	// This will allow for env vars like RCON_PORT
	viper.SetEnvPrefix("rcon")

	viper.SetConfigName(".rcon-cli") // name of config file (without extension)
	viper.AddConfigPath("$HOME")     // adding home directory as first search path
	viper.AutomaticEnv()             // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
