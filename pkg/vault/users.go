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

package vault

import (
	"net/http"
	"fmt"
	"strings"

	"github.com/gambol99/vaultctl/pkg/api"
)

// AddUser adds a user to vault
func (r *Client) AddUser(user *api.User) error {
	var params interface{}
	// step: set the path
	uri := user.Path
	// step: provision the type
	if user.UserPass != nil {
		if err := user.UserPass.IsValid(); err != nil {
			return err
		}
		// step: use the path or default to the type
		path := "userpass"
		if user.Path != "" {
			path = user.Path
		}
		uri = fmt.Sprintf("auth/%s/users/%s", path, user.UserPass.Username)
		var config struct {
			Password string `json:"password"`
			Policies string `json:"policies"`
		}
		config.Password = user.UserPass.Password
		config.Policies = strings.Join(user.Policies, ",")
		params = &config
	}

	resp, err := r.Request("PUT", uri, &params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unable to add user: %s", resp.Body)
	}

	return nil
}