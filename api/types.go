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

import "time"

// Config is the definition for a config file
type Config struct {
	// Users is a series of users
	Users []*User `yaml:"users" json:"users" hcl:"users"`
	// Backends is a series of backends
	Backends []*Backend `yaml:"backends" json:"backends" hcl:"backends"`
	// Secrets is a series of secrets
	Secrets []*Secret `yaml:"secrets" json:"secrets" hcl:"secrets"`
}

// User is the definition for a user
type User struct {
	// UserPass is the credentials for a userpass auth backend
	UserPass *UserCredentials `yaml:"userpass" json:"userpass" hcl:"userpass"`
	// Policies is a list of policies the user has access to
	Policies []string `yaml:"policies" json:"policies" hcl:"policies"`
}

// Secret defines a secret
type Secret struct {
	// Path is key for this secret
	Path string `yaml:"path" json:"path" hcl:"path"`
	// Values is a series of values associated to the secret
	Values map[string]interface{} `yaml:"values" json:"values" hcl:"values"`
}

// BackendConfig is a map of backend configuration
type BackendConfig map[string]string

// Backend defined the type and configuration for a backend in vault
type Backend struct {
	// Path is the mountpoint for the mount
	Path string `yaml:"path" json:"path" hcl:"path"`
	// Description is the a description for the backend
	Description string `yaml:"description" json:"description" hcl:"description"`
	// Type is the type of backend
	Type string `yaml:"type" json:"type" hcl:"type"`
	// DefaultLeaseTTL is the default lease of the backend
	DefaultLeaseTTL time.Duration `yaml:"default-lease-ttl" json:"default-lease-ttl" hcl:"default-lease-ttl"`
	// MaxLeaseTTL is the max ttl
	MaxLeaseTTL time.Duration `yaml:"max-lease-ttl" json:"max-lease-ttl" hcl:"max-lease-ttl"`
	// Config is the configuration of the mountpoint
	Config []*BackendConfig `yaml:"config" json:"config" hcl:"config"`
}

// UserCredentials are the userpass credentials
type UserCredentials struct {
	// Username is the id of the user
	Username string `yaml:"username" json:"username" hcl:"username"`
	// Password is the password of the user
	Password string `yaml:"password" json:"password" hcl:"password"`
}
