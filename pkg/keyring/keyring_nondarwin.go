// +build !darwin

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
package keyring

import "fmt"

// GenerateKeyringID generates a keyring-compliant key
func GenerateKeyringID(id string) string {
	return fmt.Sprintf("%s-%s", keyringSecretPrefixV1, id)
}

// GetKeyring fetches a secret from the keyring
func GetKeyring(id string) (string, Error) {
	return getKeyring(id)
}

// SetKeyring saves a value to the keyring
func SetKeyring(key string, value string) Error {
	return setKeyring(key, value)
}

// DeleteKeyring removes a value from the keyring
func DeleteKeyring(key string) Error {
	return deleteKeyring(key)
}

func ClearKeyring() Error {
	// not implemented
	return Error{}
}

func isV1Secret(key string) bool {
	return true
}
