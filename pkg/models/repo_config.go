/*
Copyright © 2020 Doppler <support@doppler.com>

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

package models

// Config struct represents the basic project setup values
type ProjectConfig struct {
	Config  string `yaml:"config"`
	Project string `yaml:"project"`
	Path    string `yaml:"path"`
}

// RepoConfig struct representing legacy doppler.yaml setup file format
// that only supported a single project and config
type RepoConfig struct {
	Setup ProjectConfig `yaml:"setup"`
}

// MultiRepoConfig struct supports doppler.yaml files containing multiple
// project and config combos
type MultiRepoConfig struct {
	Setup []ProjectConfig `yaml:"setup"`
}
