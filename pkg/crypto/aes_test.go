/*
Copyright © 2022 Doppler <support@doppler.com>

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
package crypto

import (
	"fmt"
	"testing"
)

const originalPassphrase = "secret"
const originalPlaintext = "{\"TEST_SECRET\":\"value\"}"

func TestDecrypt(t *testing.T) {
	// decode hex w/o prefix
	ciphertext := "9bc0a6db97dadea4-0d16d53716f505651f894aba-11b04a80eafd8ea700c7755de860aeb0347cff4ae93b626e858681e7e123034b4c11691a412843"
	plaintext, err := Decrypt(originalPassphrase, []byte(ciphertext), "hex")
	if err != nil || plaintext != originalPlaintext {
		t.Error("Invalid plaintext when decrypting non-prefixed hex value")
	}

	// decode hex w/ prefix
	ciphertext = fmt.Sprintf("hex:%s", ciphertext)
	plaintext, err = Decrypt(originalPassphrase, []byte(ciphertext), "hex")
	if err != nil || plaintext != originalPlaintext {
		t.Error("Invalid plaintext when decrypting hex value")
	}

	// decode base64 w/o prefix (should error)
	ciphertext = "qwbkFMWB7FE=-Ew968YdkAXRb6l46-eA4o9Pf9mSIaOofa8YIEP+FqJ6DwScHsYIObAw3dvKvHbe5SDTzB"
	plaintext, err = Decrypt(originalPassphrase, []byte(ciphertext), "base64")
	if err == nil {
		t.Error("Expected error when decrypting non-prefixed base64 value")
	}

	// decode base64 w/ prefix
	ciphertext = fmt.Sprintf("base64:%s", ciphertext)
	plaintext, err = Decrypt(originalPassphrase, []byte(ciphertext), "base64")
	if err != nil || plaintext != originalPlaintext {
		t.Error("Invalid plaintext when decrypting base64 value")
	}
}

func TestEncrypt(t *testing.T) {
	// hex
	ciphertext, err := Encrypt(originalPassphrase, []byte(originalPlaintext), "hex")
	if err != nil {
		t.Error("Invalid ciphertext when encrypting value w/ hex encoding")
	}
	plaintext, err := Decrypt(originalPassphrase, []byte(ciphertext), "hex")
	if err != nil || plaintext != originalPlaintext {
		t.Error("Invalid plaintext when decrypting hex value")
	}

	// base64
	ciphertext, err = Encrypt(originalPassphrase, []byte(originalPlaintext), "base64")
	if err != nil {
		t.Error("Invalid ciphertext when encrypting value w/ base64 encoding")
	}
	plaintext, err = Decrypt(originalPassphrase, []byte(ciphertext), "base64")
	if err != nil || plaintext != originalPlaintext {
		t.Error("Invalid plaintext when decrypting base64 value")
	}
}
