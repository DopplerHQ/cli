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
	"strconv"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/utils"
	"golang.org/x/crypto/pbkdf2"
)

const base64EncodingPrefix = "base64"
const hexEncodingPrefix = "hex"

const pbkdf2Rounds = 50000

func deriveKey(passphrase string, salt []byte, numRounds int) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, 8)
		// http://www.ietf.org/rfc/rfc2898.txt
		// Salt.
		_, err := rand.Read(salt)
		if err != nil {
			return nil, nil, err
		}
	}

	if numRounds < 0 {
		return nil, nil, errors.New("Invalid number of key derivation rounds")
	}

	return pbkdf2.Key([]byte(passphrase), salt, numRounds, 32, sha256.New), salt, nil
}

// Encrypt plaintext with a passphrase; uses pbkdf2 for key deriv and aes-256-gcm for encryption
func Encrypt(passphrase string, plaintext []byte, encoding string) (string, error) {
	now := time.Now()
	key, salt, err := deriveKey(passphrase, nil, pbkdf2Rounds)
	if err != nil {
		return "", err
	}

	utils.LogDebug(fmt.Sprintf("PBKDF2 key derivation took %d ms", time.Now().Sub(now).Milliseconds()))

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

	var encodedSalt string
	var encodedIV string
	var encodedData string
	if encoding == "base64" {
		encodedSalt = base64.StdEncoding.EncodeToString(salt)
		encodedIV = base64.StdEncoding.EncodeToString(iv)
		encodedData = base64.StdEncoding.EncodeToString(data)
	} else if encoding == "hex" {
		encodedSalt = hex.EncodeToString(salt)
		encodedIV = hex.EncodeToString(iv)
		encodedData = hex.EncodeToString(data)
	} else {
		return "", errors.New("Invalid encoding, must be one of [base64, hex]")
	}

	s := fmt.Sprintf("%s:%d:%s-%s-%s", encoding, pbkdf2Rounds, encodedSalt, encodedIV, encodedData)
	return s, nil
}

func decodeBase64(passphrase string, ciphertext string) ([]byte, []byte, []byte, error) {
	arr := strings.SplitN(ciphertext, "-", 3)
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

// Decrypt ciphertext with a passphrase.
// Formats:
// 1) `encoding:numRounds:text`
// 2) `encoding:text`
// 3) `text`
func Decrypt(passphrase string, ciphertext []byte) (string, error) {
	var salt []byte
	var iv []byte
	var data []byte

	cParts := strings.SplitN(string(ciphertext), ":", 3)
	rawEncoding := ""
	rawNumRounds := ""
	ciphertextData := ""
	if len(cParts) == 3 {
		rawEncoding = cParts[0]
		rawNumRounds = cParts[1]
		ciphertextData = cParts[2]
	} else if len(cParts) == 2 {
		rawEncoding = cParts[0]
		ciphertextData = cParts[1]
	} else if len(cParts) == 1 {
		ciphertextData = cParts[0]
	} else {
		return "", errors.New("Invalid ciphertext")
	}

	var encoding string
	if rawEncoding == base64EncodingPrefix {
		encoding = base64EncodingPrefix
	} else if rawEncoding == hexEncodingPrefix || rawEncoding == "" {
		// default to hex for backwards compatibility b/c we didn't always include an encoding prefix
		// TODO remove support for optional prefix when releasing CLI v4 (DPLR-435)
		encoding = hexEncodingPrefix
	} else {
		return "", errors.New("Invalid encoding, must be one of [base64, hex]")
	}

	numPbkdf2Rounds := pbkdf2Rounds
	if rawNumRounds != "" {
		n, err := strconv.ParseInt(rawNumRounds, 10, 32)
		if err != nil {
			return "", errors.New("Unable to parse number of rounds")
		}

		numPbkdf2Rounds = int(n)
	}

	if encoding == base64EncodingPrefix {
		var err error
		salt, iv, data, err = decodeBase64(passphrase, ciphertextData)
		if err != nil {
			return "", err
		}
	} else {
		var err error
		salt, iv, data, err = decodeHex(passphrase, ciphertextData)
		if err != nil {
			return "", err
		}
	}

	key, _, err := deriveKey(passphrase, salt, numPbkdf2Rounds)
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
