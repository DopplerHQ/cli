/*
Copyright © 2021 Doppler <support@doppler.com>

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
package keyring

import (
	"fmt"
	"strings"

	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/zalando/go-keyring"
)

const keyringService = "doppler-cli"
const keyringSecretPrefixV1 = "secret"
const keyringSecretPrefixV2 = "secret2"

// TODO update once app is signed/notarized
const keyringAccessGroup = "com.doppler.cli"

// IsKeyringSecret checks whether the secret is stored in keyring
func IsKeyringSecret(value string) bool {
	return strings.HasPrefix(value, fmt.Sprintf("%s-", keyringSecretPrefixV1)) || strings.HasPrefix(value, fmt.Sprintf("%s-", keyringSecretPrefixV2))
}

// Error keyring errors
type Error struct {
	Err     error
	Message string
}

// Unwrap get the original error
func (e *Error) Unwrap() error { return e.Err }

// IsNil whether the error is nil
func (e *Error) IsNil() bool { return e.Err == nil && e.Message == "" }

func getKeyring(id string) (string, Error) {
	value, err := keyring.Get(keyringService, id)
	if err != nil {
		if err == keyring.ErrUnsupportedPlatform {
			return "", Error{Err: err, Message: "Your OS does not support keyring"}
		} else if err == keyring.ErrNotFound {
			return "", Error{Err: err, Message: "Token not found in system keyring"}
		} else {
			return "", Error{Err: err, Message: "Unable to retrieve value from system keyring"}
		}
	}

	return value, Error{}
}

func setKeyring(key string, value string) Error {
	if err := keyring.Set(keyringService, key, value); err != nil {
		if err == keyring.ErrUnsupportedPlatform {
			return Error{Err: err, Message: "Your OS does not support keyring"}
		} else {
			return Error{Err: err, Message: "Unable to access system keyring for secure storage"}
		}
	}

	return Error{}
}

func deleteKeyring(key string) Error {
	if err := keyring.Delete(keyringService, key); err != nil {
		return Error{Err: err, Message: "Unable to remove value from keyring"}
	}

	return Error{}
}

func MigrateToken(token string) string {
	if !isV1Secret(token) || !utils.IsMacOS() {
		return ""
	}

	utils.LogDebug(fmt.Sprintf("Migrating token %s", token))
	rawToken, err := GetKeyring(token)
	if err != (Error{}) {
		utils.HandleError(err.Unwrap(), "Unable to migrate tokens to new keychain library")
	}

	uuid, e := utils.UUID()
	if e != nil {
		utils.HandleError(e, "Unable to generate UUID for keyring")
	}
	id := GenerateKeyringID(uuid)
	if err := SetKeyring(id, rawToken); err != (Error{}) {
		utils.HandleError(err.Unwrap(), "Unable to migrate tokens to new keychain library")
	}

	return id
}
