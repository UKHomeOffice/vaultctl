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
	"io/ioutil"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gambol99/vaultctl/pkg/api"
	"github.com/gambol99/vaultctl/pkg/utils"
	"github.com/gambol99/vaultctl/pkg/vault"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fatih/color"
	v "github.com/hashicorp/vault/api"
)

type syncCommand struct {
	// a collection of users
	users []*api.User
	// a collection of backends
	backends []*api.Backend
	// a collection of secrets
	secrets []*api.Secret
	// a collection of polices files
	policyFiles []string
	// a list of configuration files
	configFiles []string
	// the vault transit backend
	transit string
	// the vault client
	client *vault.Client
	// whether to skip applying the users to vault
	skipUsers bool
	// whether to skip applying the backend's to vault
	skipBackends bool
	// whether to skip applying the policies to vault
	skipPolicies bool
	// whether to skip applying the secrets to vault
	skipSecrets bool
	// whether to skip on errors and continue
	skipErrors bool
}

// newSyncCommand create a new sync command
func newSyncCommand() cli.Command {
	return new(syncCommand).getCommand()
}

func (r syncCommand) action(cx *cli.Context) error {
	startTime := time.Now()
	// step: valid the command line options
	if err := r.validateAction(cx); err != nil {
		return err
	}
	// step: get a vault client
	client, err := getVaultClient(cx)
	if err != nil {
		return err
	}
	r.client = client

	// step: parse the configuration files
	if err := r.parseConfigFiles(); err != nil {
		return err
	}
	if !r.skipPolicies {
		if err := r.applyPolicies(r.policyFiles); err != nil {
			return err
		}
	}
	if !r.skipUsers {
		if err := r.applyUsers(r.users); err != nil {
			return err
		}
	}
	if !r.skipBackends {
		if err := r.applyBackends(r.backends); err != nil {
			return err
		}
	}
	if !r.skipSecrets {
		if err := r.applySecrets(r.secrets); err != nil {
			return err
		}
	}
	log.Infof("synchronization complete, time took: %s", time.Now().Sub(startTime).String())

	return nil
}

// applyPolicies applies the policies to a vault instance
func (r syncCommand) applyPolicies(policies []string) error {
	log.Infof("%s", color.GreenString("-> synchronizing the vault policies, %d files", len(r.policyFiles)))

	for _, p := range r.policyFiles {
		name := filepath.Base(p)
		// step: read in the content
		content, err := ioutil.ReadFile(p)
		if err != nil {
			if !r.skipErrors {
				return err
			}
			log.Warnf("unable to read the policy file: %s, error: %s, skipping", p, err)
			continue
		}
		if err := r.client.SetPolicy(name, string(content)); err != nil {
			if !r.skipErrors {
				return err
			}
			log.Warnf("unable to apply policy: %s, error: %s", name, err)
			continue
		}

		log.Infof("[policy: %s] successfully applied the policy, filename: %s", name, p)
	}

	return nil
}

// applyUsers synchronizes the users with vault
func (r syncCommand) applyUsers(users []*api.User) error {
	log.Infof("%s", color.GreenString("-> synchronizing the vault users, users: %d", len(users)))

	for _, x := range users {
		// step: validate the user
		err := x.IsValid()
		if err != nil && r.skipErrors {
			log.Warningf("the user is invalid, error: %s", err)
		} else if err != nil {
			return err
		}
		log.Infof("[user: %s] ensuring user, policies: %s", x.UserPass.Username, x.GetPolicies())

		// step: attempt to add the user
		if err := r.client.AddUser(x); err != nil {
			if !r.skipErrors {
				return err
			}
			log.Warningf("[user: %s] failed to add the user, error: %s", x, err)
		}
	}

	return nil
}

// syncBackends synchronizes the backend's with vault
func (r syncCommand) applyBackends(backends []*api.Backend) error {
	log.Infof("%s", color.GreenString("-> synchronizing the backends, backend: %d", len(backends)))

	for _, backend := range backends {
		if err := backend.IsValid(); err != nil {
			if !r.skipErrors {
				return err
			}
			log.Warningf("backend is invalid: %s", err)
			continue
		}
		// step: get the backend path
		path := backend.GetPath()

		// step: get a list of mounts
		mounted, err := r.client.Mounts()
		if err != nil {
			return err
		}

		// step: check if the backend if already mounted
		_, found := mounted[backend.GetPath()+"/"]
		if !found {
			log.Infof("[backend: %s] creating backend", path)
			if err := r.client.Client().Sys().Mount(path, &v.MountInput{
				Type:        backend.Type,
				Description: backend.Description,
				Config: v.MountConfigInput{
					DefaultLeaseTTL: backend.DefaultLeaseTTL.String(),
					MaxLeaseTTL:     backend.MaxLeaseTTL.String(),
				},
			}); err != nil {
				if !r.skipErrors {
					return err
				}
				log.Warningf("unable to mount backend: %s, error: %s", path, err)
				continue
			}
		} else {
			log.Infof("[backend: %s]: already exist, moving to configuration", path)
		}

		// step: apply the configuration
		for _, c := range backend.Config {
			// step: get the path
			uri := c.GetPath(path)

			// step: check if a once type setting?
			_, oneshot := c.Map()["oneshot"]
			if found && oneshot {
				log.Infof("[backend:%s] skipping the config, as it's a oneshot setting", uri)
				continue
			}

			log.Infof("[backend->config: %s] applying configuration for backend", uri)

			resp, err := r.client.Request("PUT", uri, &c)
			if err != nil {
				if !r.skipErrors {
					return err
				}
				log.Warningf("unable to mount backend: %s, error: %s", path, err)
				continue
			}
			if resp.StatusCode != http.StatusNoContent {
				if !r.skipErrors {
					return err
				}
				log.Warningf("unable to apply config: %s, error: %s", c.URI(), resp.Body)
			}
		}
	}

	return nil
}

// applySecrets synchronizes the secrets in vault
func (r *syncCommand) applySecrets(secrets []*api.Secret) error {
	log.Infof("%s", color.GreenString("-> synchronizing the secrets with vault, secrets: %d", len(secrets)))
	for _, s := range secrets {
		// step: validate the secret
		if err := s.IsValid(); err != nil {
			if r.skipErrors {
				log.Warningf("secret invalid, error: %s", err)
				continue
			}
			return err
		}

		log.Infof("[secret: %s] adding the secret", s.Path)

		// step: apply the secret
		if err := r.client.AddSecret(s); err != nil {
			if r.skipErrors {
				log.Warningf("failed to add the secret: %s, error: %s", s.Path, err)
				continue
			}
			return err
		}
	}

	return nil
}

// parseConfigFiles reads a series of configuration files or directories and extracts the
// items from them
func (r *syncCommand) parseConfigFiles() error {
	// step: iterate the configuration files and decode
	for _, c := range r.configFiles {
		cfg := new(api.Config)

		if err := utils.DecodeFile(c, cfg); err != nil {
			return fmt.Errorf("unable to decode the file: %s, error: %s", c, err)
		}

		// step: appends the elements
		r.users = append(r.users, cfg.Users...)
		r.backends = append(r.backends, cfg.Backends...)
		r.secrets = append(r.secrets, cfg.Secrets...)
	}

	return nil
}

// validateAction validates the inputs from the command line
func (r *syncCommand) validateAction(cx *cli.Context) error {
	r.skipUsers = cx.Bool("skip-users")
	r.skipPolicies = cx.Bool("skip-policies")
	r.skipBackends = cx.Bool("skip-backends")
	r.skipSecrets = cx.Bool("skip-secrets")
	r.skipErrors = cx.Bool("skip-errors")
	r.configFiles = cx.StringSlice("config")
	r.transit = cx.String("transit")

	// step: check the skips
	if r.skipBackends && r.skipPolicies && r.skipUsers {
		return fmt.Errorf("you are skipping all the resources, what exactly are we syncing")
	}
	if r.skipBackends {
		log.Infof("skipping the synchronization of backends")
	}
	if r.skipPolicies {
		log.Infof("skipping the synchronization of policies")
	}
	if r.skipUsers {
		log.Infof("skipping the synchronization of users")
	}

	// step: get the files from any config directories
	files, err := utils.FindFilesInDirectory(cx.StringSlice("config-dir"), cx.String("config-extension"))
	if err != nil {
		return err
	}
	r.configFiles = append(r.configFiles, files...)

	// step: get a list of policy files from the directories
	r.policyFiles, err = utils.FindFilesInDirectory(cx.StringSlice("policies"), cx.String("policy-extension"))
	if err != nil {
		return err
	}

	return nil
}

// getCommand returns the command set
func (r syncCommand) getCommand() cli.Command {
	return cli.Command{
		Name:    "synchronize",
		Aliases: []string{"sync"},
		Usage:   "synchonrize the users, policies, secrets and backends",
		Action: func(cx *cli.Context) {
			executeCommand(cx, r.action)
		},
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "c, config",
				Usage: "the path to a configuration file containing users, backends and or secrets",
			},
			cli.StringSliceFlag{
				Name:  "C, config-dir",
				Usage: "the path to a directory containing one of more config files",
			},
			cli.StringSliceFlag{
				Name:  "p, policies",
				Usage: "the path to a directory containing one of more policy files",
			},
			cli.StringFlag{
				Name:  "t, transit",
				Usage: "the vault transit endpoint we should use to decrypt the files",
			},
			cli.BoolFlag{
				Name:  "skip-policies",
				Usage: "wheather or not to skip synchronizing the policies",
			},
			cli.BoolFlag{
				Name:  "skip-users",
				Usage: "wheather or not to skip synchronizing the vault users",
			},
			cli.BoolFlag{
				Name:  "skip-backends",
				Usage: "wheather or not to skip synchronizing the backends",
			},
			cli.BoolFlag{
				Name:  "skip-secrets",
				Usage: "wheather or not to skip synchronizing the secrets",
			},
			cli.BoolFlag{
				Name:  "skip-error",
				Usage: "wheather or not to skip errors and attempt to finish regardless",
			},
			cli.StringFlag{
				Name:  "policy-extension",
				Usage: "the file extenions of the policy files",
				Value: "*.hcl",
			},
			cli.StringFlag{
				Name:  "config-extension",
				Usage: "when using a config-dir, the file extension to glob",
				Value: "*.yml",
			},
		},
	}
}
