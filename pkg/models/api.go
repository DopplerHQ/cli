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

type SecretValueType struct {
	Type string `json:"type"`
}

// ComputedSecret holds all info about a secret
type ComputedSecret struct {
	Name               string          `json:"name"`
	RawValue           *string         `json:"raw"`
	ComputedValue      *string         `json:"computed"`
	RawVisibility      string          `json:"rawVisibility"`
	ComputedVisibility string          `json:"computedVisibility"`
	RawValueType       SecretValueType `json:"rawValueType"`
	ComputedValueType  SecretValueType `json:"computedValueType"`
	Note               string          `json:"note"`
}

// ChangeRequest can be used to smartly update secrets
type ChangeRequest struct {
	Name               string           `json:"name"`
	OriginalName       interface{}      `json:"originalName"`
	Value              interface{}      `json:"value"`
	OriginalValue      interface{}      `json:"originalValue,omitempty"`
	Visibility         *string          `json:"visibility,omitempty"`
	OriginalVisibility *string          `json:"originalVisibility,omitempty"`
	ValueType          *SecretValueType `json:"valueType,omitempty"`
	OriginalValueType  *SecretValueType `json:"originalValueType,omitempty"`
	ShouldPromote      *bool            `json:"shouldPromote,omitempty"`
	ShouldDelete       *bool            `json:"shouldDelete,omitempty"`
	ShouldConverge     *bool            `json:"shouldConverge,omitempty"`
}

// SecretNote contains a secret and its note
type SecretNote struct {
	Secret string `json:"secret"`
	Note   string `json:"note"`
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
}

// EnvironmentInfo environment info
type EnvironmentInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CreatedAt      string `json:"created_at"`
	InitialFetchAt string `json:"initial_fetch_at"`
	Project        string `json:"project"`
}

// ConfigInfo project info
type ConfigInfo struct {
	Name           string             `json:"name"`
	Root           bool               `json:"root"`
	Locked         bool               `json:"locked"`
	Environment    string             `json:"environment"`
	Project        string             `json:"project"`
	CreatedAt      string             `json:"created_at"`
	InitialFetchAt string             `json:"initial_fetch_at"`
	LastFetchAt    string             `json:"last_fetch_at"`
	Inheritable    bool               `json:"inheritable"`
	Inherits       []ConfigDescriptor `json:"inherits"`
	InheritedBy    []ConfigDescriptor `json:"inheritedBy"`
}

// ConfigLog a log
type ConfigLog struct {
	ID          string    `json:"id"`
	Text        string    `json:"text"`
	HTML        string    `json:"html"`
	CreatedAt   string    `json:"created_at"`
	Config      string    `json:"config"`
	Environment string    `json:"environment"`
	Project     string    `json:"project"`
	User        User      `json:"user"`
	Diff        []LogDiff `json:"diff"`
}

// ActivityLog an activity log
type ActivityLog struct {
	ID                 string `json:"id"`
	Text               string `json:"text"`
	HTML               string `json:"html"`
	CreatedAt          string `json:"created_at"`
	EnclaveConfig      string `json:"enclave_config"`
	EnclaveEnvironment string `json:"enclave_environment"`
	EnclaveProject     string `json:"enclave_project"`
	User               User   `json:"user"`
}

// User user profile
type User struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	Username     string `json:"username"`
	ProfileImage string `json:"profile_image_url"`
}

// LogDiff diff of log entries
type LogDiff struct {
	Name    string `json:"name"`
	Added   string `json:"added"`
	Removed string `json:"removed"`
}

// ConfigServiceToken a service token
type ConfigServiceToken struct {
	Name        string `json:"name"`
	Token       string `json:"token"`
	Slug        string `json:"slug"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at"`
	Project     string `json:"project"`
	Environment string `json:"environment"`
	Config      string `json:"config"`
	Access      string `json:"access"`
}

// APISecretResponse is the response the secrets endpoint returns
type APISecretResponse struct {
	Success bool                 `json:"success"`
	Secrets map[string]APISecret `json:"secrets"`
}

// APISecret is the object the API returns for a given secret
type APISecret struct {
	RawValue           *string         `json:"raw"`
	ComputedValue      *string         `json:"computed"`
	RawVisibility      string          `json:"rawVisibility"`
	ComputedVisibility string          `json:"computedVisibility"`
	RawValueType       SecretValueType `json:"rawValueType"`
	ComputedValueType  SecretValueType `json:"computedValueType"`
	Note               string          `json:"note"`
}

type ActorInfo struct {
	Workplace    ActorWorkplaceInfo `json:"workplace"`
	Type         string             `json:"type"`
	TokenPreview string             `json:"token_preview"`
	Slug         string             `json:"slug"`
	CreatedAt    string             `json:"created_at"`
	Name         string             `json:"name"`
	LastSeenAt   string             `json:"last_seen_at"`
}
type ActorWorkplaceInfo struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type WatchSecrets struct {
	Type string `json:"type"`
}

type ConfigDescriptor struct {
	Project string `json:"project"`
	Config  string `json:"config"`
}
