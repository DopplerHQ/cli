/*
Copyright © 2019 Doppler <support@doppler.com>

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

// ComputedSecret holds computed and raw value
type ComputedSecret struct {
	Name          string `json:"name"`
	RawValue      string `json:"raw"`
	ComputedValue string `json:"computed"`
}

// WorkplaceSettings workplace settings
type WorkplaceSettings struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	BillingEmail string `json:"billing_email"`
}

// ProjectInfo project info
type ProjectInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	SetupAt     string `json:"setup_at"`
}

// EnvironmentInfo environment info
type EnvironmentInfo struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	CreatedAt        string   `json:"created_at"`
	FirstDeployAt    string   `json:"first_deploy_at"`
	SetupAt          string   `json:"setup_at"`
	Project          string   `json:"pipeline"`
	MissingVariables []string `json:"missing_variables"`
}

// ConfigInfo project info
type ConfigInfo struct {
	Name             string   `json:"name"`
	Environment      string   `json:"stage"`
	Project          string   `json:"project"`
	CreatedAt        string   `json:"created_at"`
	DeployedAt       string   `json:"deployed_at"`
	MissingVariables []string `json:"missing_variables"`
}

// Log a log
type Log struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	HTML        string `json:"html"`
	CreatedAt   string `json:"created_at"`
	Config      string `json:"environment"`
	Environment string `json:"stage"`
	Project     string `json:"pipeline"`
	User        User   `json:"user"`
}

// User user profile
type User struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	Username     string `json:"username"`
	ProfileImage string `json:"profile_image_url"`
}
