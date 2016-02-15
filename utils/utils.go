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

package utils

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// DecodeConfig unmarshal's the configuration file
func DecodeConfig(path string, data interface{}) error {
	// step: read in the content
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	switch filepath.Ext(path) {
	case ".json":
		err = json.Unmarshal(content, data)
	case ".yml":
		fallthrough
	case ".yaml":
		err = yaml.Unmarshal(content, data)
	default:
		return fmt.Errorf("unsupported config file content, extension: %s", filepath.Ext(path))
	}

	if err != nil {
		return err
	}

	return nil
}

// ContainedIn checks if a value in a list of a strings
func ContainedIn(value string, list []string) bool {
	for _, x := range list {
		if x == value {
			return true
		}
	}

	return false
}

// IsDirectory checks if a path is a directory
func IsDirectory(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	return stat.IsDir()
}

// IsFile checks if the path is a file
func IsFile(path string) bool {
	return !IsDirectory(path)
}
