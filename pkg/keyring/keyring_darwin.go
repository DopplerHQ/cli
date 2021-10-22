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
	"github.com/keybase/go-keychain"
)

// GenerateKeyringID generates a keyring-compliant key
func GenerateKeyringID(id string) string {
	return fmt.Sprintf("%s-%s", keyringSecretPrefixV2, id)
}

// GetKeyring fetches a secret from the keyring
func GetKeyring(key string) (string, Error) {
	if strings.HasPrefix(key, keyringSecretPrefixV2) {
		item, err := keychain.GetGenericPassword(keyringService, key, "", keyringAccessGroup)

		if err != nil {
			return "", Error{Err: err, Message: "Unable to retrieve value from system keyring"}
		}

		if item == nil && err == nil {
			return "", Error{Err: err, Message: "Token not found in system keyring"}
		}

		return string(item), Error{}
	}

	return getKeyring(key)
}

// SetKeyring saves a value to the keyring
func SetKeyring(key string, value string) Error {
	item := keychain.NewGenericPassword(keyringService, key, "", []byte(value), keyringAccessGroup)
	item.SetAccount(key)
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	if err := keychain.AddItem(item); err != nil {
		return Error{Err: err, Message: "Unable to access system keyring for secure storage"}
	}

	return Error{}
}

// DeleteKeyring removes a value from the keyring
func DeleteKeyring(key string) Error {
	if strings.HasPrefix(key, keyringSecretPrefixV2) {
		if err := keychain.DeleteGenericPasswordItem(keyringService, key); err != nil {
			return Error{Err: err, Message: "Unable to remove value from keyring"}
		}
	} else {
		return deleteKeyring(key)
	}

	return Error{}
}

func ClearKeyring() Error {
	accounts, err := keychain.GetAccountsForService(keyringService)
	if err != nil {
		return Error{Err: err, Message: "Unable to access system keyring"}
	}

	for _, account := range accounts {
		if err := keychain.DeleteGenericPasswordItem(keyringService, account); err != nil {
			// log errors and continue executing
			utils.LogDebugError(err)
		}
	}

	return Error{}
}

func isV1Secret(key string) bool {
	return strings.HasPrefix(key, fmt.Sprintf("%s-", keyringSecretPrefixV1))
}
