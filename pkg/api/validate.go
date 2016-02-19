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

package api

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gambol99/vaultctl/pkg/utils"
)

var (
	supportedBackendTypes = []string{
		"aws", "generic", "pki", "transit",
		"cassandra", "consul", "cubbyhole", "mysql",
		"postgres", "ssh", "custom",
	}

	supportAuthTypes = []string{
		"userpass", "ldap", "token", "appid", "github", "mfa", "tls",
	}
)

// IsValid validates the attributes
func (r Attributes) IsValid() error {
	if r.URI() == "" {
		return fmt.Errorf("attributes must have a uri specified")
	}

	return nil
}

// IsValid validates the auth backend
func (r Auth) IsValid() error {
	if r.Type == "" {
		return fmt.Errorf("you must specify a auth type")
	}
	if r.Path == "" {
		return fmt.Errorf("you must specify a path")
	}
	if strings.HasSuffix(r.Path, "/") {
		return fmt.Errorf("path should not end with /")
	}
	if !utils.ContainedIn(r.Type, supportAuthTypes) {
		return fmt.Errorf("auth type: %s is a unsupported auth type", r.Type)
	}

	for i, x := range r.Attrs {
		if err := x.IsValid(); err != nil {
			return fmt.Errorf("attribute %s invalid, error: %s", i, err)
		}
	}

	return nil
}

// IsValid validates the user is ok
func (r *User) IsValid() error {
	if r.Path != "" && strings.HasSuffix(r.Path, "/") {
		return fmt.Errorf("path should not end with /")
	}

	if r.UserPass != nil {
		return r.UserPass.IsValid()
	}

	return fmt.Errorf("you have not added authentication to the user")
}

// IsValid validates the user credential is ok
func (r UserCredentials) IsValid() error {
	if r.Username == "" {
		return fmt.Errorf("does not have a username")
	}
	if r.Password == "" {
		return fmt.Errorf("does not have a password")
	}

	return nil
}

// IsValid validates the secret is ok
func (r Secret) IsValid() error {
	if r.Path == "" {
		return fmt.Errorf("the secret must have a path")
	}
	if r.Values == nil || len(r.Values) <= 0 {
		return fmt.Errorf("the secret must have some values")
	}

	return nil
}

// IsValid validates the backend is ok
func (r Backend) IsValid() error {
	if r.Path == "" {
		return fmt.Errorf("backend must have a path")
	}
	if r.Type == "" {
		return fmt.Errorf("backend %s must have a type", r.Path)
	}
	if r.Description == "" {
		return fmt.Errorf("backend %s must have a description", r.Path)
	}
	if r.MaxLeaseTTL.Seconds() < r.DefaultLeaseTTL.Seconds() {
		return fmt.Errorf("backend: %s, max lease ttl cannot be less than the default", r.Path)
	}
	if r.DefaultLeaseTTL.Seconds() < 0 {
		return fmt.Errorf("backend: %s, default lease time must be positive", r.Path)
	}
	if r.MaxLeaseTTL.Seconds() < 0 {
		return fmt.Errorf("backend: %s, max lease time must be positive", r.Path)
	}
	if !utils.ContainedIn(r.Type, supportedBackendTypes) {
		return fmt.Errorf("backend: %s, unsupported type: %s, supported types are: %s", r.Path, r.Type, supportedBackends())
	}
	if r.Attrs != nil && len(r.Attrs) > 0 {
		for _, x := range r.Attrs {
			// step: ensure the config has a uri
			if x.URI() == "" {
				return fmt.Errorf("backend: %s, config for must have uri", r.Path)
			}
			// step: read in a any files reference by @path
			for k, v := range x.Values() {
				if strings.HasPrefix(v, "@") {
					path := strings.TrimPrefix(v, "@")
					if !utils.IsFile(path) {
						return fmt.Errorf("backend: %s, file referenced in config: %v does not exist", r.Path, v)
					}
					// step: read in the file and update the key with the content
					content, err := ioutil.ReadFile(path)
					if err != nil {
						return err
					}
					(*x)[k] = string(content)
				}
			}
		}
	}

	return nil
}

// supportedBackends returns a list of supported backend types
func supportedBackends() string {
	return strings.Join(supportedBackendTypes, ",")
}
