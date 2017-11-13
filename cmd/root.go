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

	"github.com/jshirley/slack-ask/asker"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	oauth        string
	clientId     string
	secret       string
	token        string
	mongodb      string
	bind         string
	jiraEndpoint string
	jiraUsername string
	jiraPassword string
	jiraPublic   string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "slack-ask",
	Short: "An application to collect questions from Slack, and do things with them.",
	Long: `Slack is not great at structuring incoming questions, and those questions
	really require good responses. The best response, however, is fixing the underlying
	issue.
	
	Inspired by Alan Shreve's amazing blogpost "Sweat the Small Stuff" at
	https://inconshreveable.com/09-09-2014/sweat-the-small-stuff/ this is an approach
	to help people collect and categorize questions on Slack, track and aggregate,
	and prioritize underlying fixes`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetString("oauth") == "" {
			fmt.Printf("Multi-account support is not setup yet, use oauth first and install manually into the workspace")
			return
		}

		client, err := asker.NewAsker(viper.GetString("oauth"), viper.GetString("token"), viper.GetString("mongodb"))
		if err != nil {
			log.Fatal(err)
			return
		}

		var dialog asker.Dialog
		unmarshalError := viper.Unmarshal(&dialog)
		if unmarshalError == nil && len(dialog.Elements) > 0 {
			err := client.SetDialogElements(dialog)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}

		if viper.GetString("jira") != "" {
			jiraClient, err := client.NewJira(viper.GetString("jira"), viper.GetString("jirauser"), viper.GetString("jirapass"), viper.GetString("publicJira"))
			if err != nil {
				log.Fatal(err)
				return
			}
			client.Jira = jiraClient
		}
		go client.CleanQueue()
		client.Listen(viper.GetString("bind"))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.slack-ask.yaml)")
	RootCmd.PersistentFlags().StringVar(&oauth, "oauth", "", "OAuth token for single install apps")
	RootCmd.PersistentFlags().StringVar(&clientId, "client", "", "Slack Client ID")
	RootCmd.PersistentFlags().StringVar(&secret, "secret", "", "Secret")
	RootCmd.PersistentFlags().StringVar(&token, "token", "", "Slack verification token")
	RootCmd.PersistentFlags().StringVar(&mongodb, "mongodb", "localhost:27017", "Connection string for MongoDB (default is localhost:27017)")
	RootCmd.PersistentFlags().StringVar(&bind, "bind", ":3000", "Bind address to listen on (default is 0.0.0.0:3000)")

	RootCmd.PersistentFlags().StringVar(&jiraEndpoint, "jira", "", "The JIRA endpoint to use")
	RootCmd.PersistentFlags().StringVar(&jiraUsername, "jirauser", "", "The JIRA username")
	RootCmd.PersistentFlags().StringVar(&jiraPassword, "jirapass", "", "The JIRA password")
	RootCmd.PersistentFlags().StringVar(&jiraPublic, "publicJira", "", "The JIRA public endpoint (to link tickets at), you may not need this.")

	viper.BindPFlag("oauth", RootCmd.PersistentFlags().Lookup("oauth"))
	viper.BindPFlag("client", RootCmd.PersistentFlags().Lookup("client"))
	viper.BindPFlag("secret", RootCmd.PersistentFlags().Lookup("secret"))
	viper.BindPFlag("token", RootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("mongodb", RootCmd.PersistentFlags().Lookup("mongodb"))
	viper.BindPFlag("bind", RootCmd.PersistentFlags().Lookup("bind"))

	viper.BindPFlag("jira", RootCmd.PersistentFlags().Lookup("jira"))
	viper.BindPFlag("jirauser", RootCmd.PersistentFlags().Lookup("jirauser"))
	viper.BindPFlag("jirapass", RootCmd.PersistentFlags().Lookup("jirapass"))
	viper.BindPFlag("publicJira", RootCmd.PersistentFlags().Lookup("publicJira"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigType("yaml")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".slack-ask" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".slack-ask")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println(err)
		os.Exit(1)
	}
}
