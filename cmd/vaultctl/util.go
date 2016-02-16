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
	"github.com/gambol99/vaultctl/pkg/utils"
	"github.com/gambol99/vaultctl/pkg/vault"

	"github.com/codegangsta/cli"
)

// getVaultClient retrieves a vault client for use
func getVaultClient(cx *cli.Context) (*vault.Client, error) {
	host := cx.GlobalString("vault-addr")
	username := cx.GlobalString("vault-username")
	password := cx.GlobalString("vault-password")
	creds := cx.GlobalString("credentials")

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
		return nil, err
	}

	return client, nil
}

func decryptTransit(path, key, content string) {

}
