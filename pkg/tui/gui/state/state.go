/*
Copyright © 2023 Doppler <support@doppler.com>

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
package state

import (
	"time"

	"github.com/DopplerHQ/cli/pkg/models"
)

type Project struct {
	Name string
}

type Config struct {
	Name        string
	Environment string
	Root        bool
}

type Secret struct {
	Name       string
	Value      string
	Visibility string
}

type ByName []Secret

func (x ByName) Len() int           { return len(x) }
func (x ByName) Less(i, j int) bool { return x[i].Name < x[j].Name }
func (x ByName) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

type State struct {
	projects     []Project
	configs      []Config
	secrets      []Secret
	secretsSetAt int64

	active struct {
		project string
		config  string
	}

	filter  string
	changes []models.ChangeRequest
}

var state *State

func init() {
	projects := make([]Project, 0)
	configs := make([]Config, 0)
	secrets := make([]Secret, 0)

	state = &State{
		projects: projects,
		configs:  configs,
		secrets:  secrets,
	}
}

func Projects() []Project            { return state.projects }
func SetProjects(projects []Project) { state.projects = projects }

func Configs() []Config           { return state.configs }
func SetConfigs(configs []Config) { state.configs = configs }

func Secrets() []Secret   { return state.secrets }
func SecretsSetAt() int64 { return state.secretsSetAt }
func SetSecrets(secrets []Secret, projectName string, configName string) {
	state.secrets = secrets
	state.secretsSetAt = time.Now().Unix()
	state.active.project = projectName
	state.active.config = configName
}

func Active() (string, string) { return state.active.project, state.active.config }

func Filter() string          { return state.filter }
func SetFilter(filter string) { state.filter = filter }

func Changes() []models.ChangeRequest           { return state.changes }
func SetChanges(changes []models.ChangeRequest) { state.changes = changes }
