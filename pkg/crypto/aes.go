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

// From https://gist.github.com/tscholl2/dc7dc15dc132ea70a98e8542fefffa28

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/utils"
	"golang.org/x/crypto/pbkdf2"
)

const Base64EncodingPrefix = "base64:"
const HexEncodingPrefix = "hex:"

func deriveKey(passphrase string, salt []byte) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, 8)
		// http://www.ietf.org/rfc/rfc2898.txt
		// Salt.
		_, err := rand.Read(salt)
		if err != nil {
			return nil, nil, err
		}
	}

	return pbkdf2.Key([]byte(passphrase), salt, 50000, 32, sha256.New), salt, nil
}

// Encrypt plaintext with a passphrase; uses pbkdf2 for key deriv and aes-256-gcm for encryption
func Encrypt(passphrase string, plaintext []byte, encoding string) (string, error) {
	now := time.Now()
	key, salt, err := deriveKey(passphrase, nil)
	if err != nil {
		return "", err
	}

	utils.LogDebug(fmt.Sprintf("PBKDF2 key derivation took %d ms", time.Since(now).Milliseconds()))

	iv := make([]byte, 12)
	// http://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf
	// Section 8.2
	_, err = rand.Read(iv)
	if err != nil {
		return "", err
	}

	b, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(b)
	if err != nil {
		return "", err
	}

	data := aesgcm.Seal(nil, iv, plaintext, nil)

	var prefix string
	var encodedSalt string
	var encodedIV string
	var encodedData string
	switch encoding {
	case "base64":
		prefix = Base64EncodingPrefix
		encodedSalt = base64.StdEncoding.EncodeToString(salt)
		encodedIV = base64.StdEncoding.EncodeToString(iv)
		encodedData = base64.StdEncoding.EncodeToString(data)
	case "hex":
		prefix = HexEncodingPrefix
		encodedSalt = hex.EncodeToString(salt)
		encodedIV = hex.EncodeToString(iv)
		encodedData = hex.EncodeToString(data)
	default:
		return "", errors.New("Invalid encoding, must be one of [base64, hex]")
	}

	s := fmt.Sprintf("%s%s-%s-%s", prefix, encodedSalt, encodedIV, encodedData)
	return s, nil
}

func decodeBase64(passphrase string, ciphertext string) ([]byte, []byte, []byte, error) {
	// prefix is required
	if !strings.HasPrefix(ciphertext, Base64EncodingPrefix) {
		return []byte{}, []byte{}, []byte{}, errors.New("Invalid ciphertext")
	}

	arr := strings.SplitN(ciphertext[len(Base64EncodingPrefix):], "-", 3)
	if len(arr) != 3 {
		return []byte{}, []byte{}, []byte{}, errors.New("Invalid ciphertext")
	}

	var salt []byte
	var iv []byte
	var data []byte

	var err error

	salt, err = base64.StdEncoding.DecodeString(arr[0])
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}
	iv, err = base64.StdEncoding.DecodeString(arr[1])
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}
	data, err = base64.StdEncoding.DecodeString(arr[2])
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	return salt, iv, data, nil
}

func decodeHex(passphrase string, ciphertext string) ([]byte, []byte, []byte, error) {
	// prefix is optional
	// TODO make this required when releasing CLI v4 (DPLR-435)
	ciphertext = strings.TrimPrefix(ciphertext, HexEncodingPrefix)

	arr := strings.SplitN(string(ciphertext), "-", 3)
	if len(arr) != 3 {
		return []byte{}, []byte{}, []byte{}, errors.New("Invalid ciphertext")
	}

	var salt []byte
	var iv []byte
	var data []byte

	var err error

	salt, err = hex.DecodeString(arr[0])
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	iv, err = hex.DecodeString(arr[1])
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	data, err = hex.DecodeString(arr[2])
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	return salt, iv, data, nil
}

// Decrypt ciphertext with a passphrase
func Decrypt(passphrase string, ciphertext []byte, encoding string) (string, error) {
	var salt []byte
	var iv []byte
	var data []byte
	switch encoding {
	case "base64":
		var err error
		salt, iv, data, err = decodeBase64(passphrase, string(ciphertext))
		if err != nil {
			return "", err
		}
	case "hex":
		var err error
		salt, iv, data, err = decodeHex(passphrase, string(ciphertext))
		if err != nil {
			return "", err
		}
	default:
		return "", errors.New("Invalid encoding, must be one of [base64, hex]")
	}

	key, _, err := deriveKey(passphrase, salt)
	if err != nil {
		return "", err
	}

	b, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(b)
	if err != nil {
		return "", err
	}

	data, err = aesgcm.Open(nil, iv, data, nil)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
