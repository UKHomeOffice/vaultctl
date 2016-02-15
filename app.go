/*
Copyright 2015 All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	"github.com/gambol99/vaultctl/vault"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/gambol99/vaultctl/utils"
)

func newVaultConfig() *cli.App {
	// step: create and return the application
	app := cli.NewApp()
	app.Usage = "is a utility for provisioning a hashicorp's vault service"
	app.Author = Author
	app.Email = Email
	app.Version = Version
	app.Flags = getGlobalOptions()
	app.Commands = []cli.Command{
		newSyncCommand(),
	}

	return app
}

// executeCommand implement the action
func executeCommand(cx *cli.Context, action func(*cli.Context, *factory) error) {
	host := cx.GlobalString("vault-addr")
	username := cx.GlobalString("vault-username")
	password := cx.GlobalString("vault-password")
	creds := cx.GlobalString("credentials")
	verbose := cx.GlobalBool("verbose")

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	f := &factory{}

	// step: validate we have the requirements
	if creds == "" {
		if username == "" {
			printUsage("you have not specified the vault username")
		}
		if password == "" {
			printUsage("you have not specified the vault password")
		}
	} else {
		if !utils.IsFile(creds) {
			printUsage("the vault credentials file does not exist")
		}
	}

	// step: create a vault client
	client, err := vault.New(host, username, password, creds)
	if err != nil {
		printUsage(fmt.Sprintf("unable to create vault client, error: %s", err.Error()))
	}
	f.client = client

	// step: ensure we capture any panics
	defer func() {
		if r := recover(); r != nil {
			printUsage(fmt.Sprintf("%s", r))
		}
	}()

	if err := action(cx, f); err != nil {
		printUsage(err.Error())
	}
}

// getGlobalOptions retrieves the command line options
func getGlobalOptions() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   "A, vault-addr",
			Usage:  "the url address of the vault service",
			Value:  "http://127.0.0.1:8200",
			EnvVar: "VAULT_ADDR",
		},
		cli.StringFlag{
			Name:   "u, vault-username",
			Usage:  "the vault username to use to authenticate to vault service",
			EnvVar: "VAULT_USERNAME",
		},

		cli.StringFlag{
			Name:   "p, vault-password",
			Usage:  "the vault password to use to authenticate to vault service",
			EnvVar: "VAULT_PASSWORD",
		},
		cli.StringFlag{
			Name:   "c, credentials",
			Usage:  "the path to a file (json|yaml) containing the username and password for userpass authenticaion",
			EnvVar: "VAULT_CRENDENTIALS",
		},
		cli.StringFlag{
			Name:   "k, kubeconfig",
			Usage:  "the path to a file containing the kubeconfig for kubernetes authentication",
			Value:  os.Getenv("HOME") + "/.kube.config",
			EnvVar: "KUBE_CONFIG",
		},
		cli.StringFlag{
			Name:   "C, kube-context",
			Usage:  "the kube context to use for authenticating to the api",
			EnvVar: "KUBE_CONTEXT",
		},
		cli.StringFlag{
			Name:   "H, kube-api",
			Usage:  "the url for the kubernetes api used to polulate the vault secrets",
			EnvVar: "KUBE_API",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "switch on verbose logging for debug purposed",
		},
		cli.BoolFlag{
			Name:  "kube-populate",
			Usage: "whether or not to populate the vault crendentials into the namespaces",
		},
	}
}

// printUsage prints the error message
func printUsage(message string) {
	fmt.Fprintf(os.Stderr, "[error] %s\n", message)
	os.Exit(1)
}
