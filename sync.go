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

	"github.com/gambol99/vaultctl/api"
	"github.com/gambol99/vaultctl/utils"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fatih/color"
	v "github.com/hashicorp/vault/api"
)

type syncCommand struct {
	// a collection of config files
	configFiles []string
	// a collection of polices files
	policyFiles []string
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

func (r syncCommand) action(cx *cli.Context, f *factory) error {
	startTime := time.Now()

	// step: valid the command line options
	if err := r.validateAction(cx); err != nil {
		return err
	}

	// step: sync the policies
	if !r.skipPolicies {
		err := r.syncPolicies(r.policyFiles, f)
		if err != nil && !r.skipErrors {
			return err
		}
	}

	// step: read in the configuration files
	for _, c := range r.configFiles {
		config := new(api.Config)
		// step: decode the configuration file
		err := utils.DecodeConfig(c, config)
		if err != nil && !r.skipErrors {
			return fmt.Errorf("failed to parse the configuration file: %s, error: %s", c, err)
		}
		if !r.skipUsers && len(config.Users) > 0 {
			if err := r.syncUsers(config.Users, f); err != nil {
				return err
			}
		}
		if !r.skipBackends && len(config.Backends) > 0 {
			if err := r.syncBackends(config.Backends, f); err != nil {
				return err
			}
		}
		if !r.skipSecrets && len(config.Secrets) > 0 {
			if err := r.syncSecrets(config.Secrets, f); err != nil {
				return err
			}
		}
	}

	log.Infof("synchronization complete, time took: %s", time.Now().Sub(startTime).String())

	return nil
}

func (r syncCommand) syncPolicies(policies []string, f *factory) error {
	log.Infof("%s", color.GreenString("-> synchronizing the vault policies"))

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
		if err := f.client.SetPolicy(name, string(content)); err != nil {
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

// syncUsers synchronizes the
func (r syncCommand) syncUsers(users []*api.User, f *factory) error {
	log.Infof("%s", color.GreenString("-> synchronizing the vault users"))

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
		if err := f.client.AddUser(x); err != nil {
			if !r.skipErrors {
				return err
			}
			log.Warningf("[user: %s] failed to add the user, error: %s", x, err)
		}
	}

	return nil
}

// syncBackends synchronizes the backend's with vault
func (r syncCommand) syncBackends(backends []*api.Backend, f *factory) error {
	log.Infof("%s", color.GreenString("-> synchronizing the backends"))

	for _, backend := range backends {
		// step: validate the backend
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
		mounted, err := f.client.Mounts()
		if err != nil {
			return err
		}

		// step: check if the backend if already mounted
		_, found := mounted[backend.GetPath()+"/"]
		if !found {
			log.Infof("[backend: %s] creating backend", path)
			if err := f.client.Client().Sys().Mount(path, &v.MountInput{
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

			resp, err := f.client.Request("PUT", uri, &c)
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

// syncSecrets synchronizes the secrets in vault
func (r *syncCommand) syncSecrets(secrets []*api.Secret, f *factory) error {
	log.Infof("%s", color.GreenString("-> synchronizing the secrets with vault"))
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
		if err := f.client.AddSecret(s); err != nil {
			if r.skipErrors {
				log.Warningf("failed to add the secret: %s, error: %s", s.Path, err)
				continue
			}
			return err
		}
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

	// step: check the config directories
	for _, x := range cx.StringSlice("config-dir") {
		if !utils.IsDirectory(x) {
			return fmt.Errorf("the path %s is not a directory", x)
		}
		files, err := filepath.Glob(fmt.Sprintf("%s/%s", x, cx.String("config-extension")))
		if err != nil {
			return err
		}
		for _, j := range files {
			if !utils.IsFile(j) {
				continue
			}
			r.configFiles = append(r.configFiles, j)
		}

	}

	// step: check the config directories
	for _, x := range cx.StringSlice("policies") {
		if !utils.IsDirectory(x) {
			return fmt.Errorf("the path %s is not a directory", x)
		}
		files, err := filepath.Glob(fmt.Sprintf("%s/%s", x, cx.String("policy-extension")))
		if err != nil {
			return err
		}
		log.Infof("found %d files under policies directory: %s", len(files), x)
		for _, j := range files {
			if !utils.IsFile(j) {
				continue
			}
			r.policyFiles = append(r.policyFiles, j)
		}
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
