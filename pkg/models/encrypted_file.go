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
package models

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/DopplerHQ/cli/pkg/utils"
)

const Pbkdf2Rounds = 500000
const LegacyPbkdf2Rounds = 50000

const Base64EncodingPrefix = "base64"
const HexEncodingPrefix = "hex"

type EncryptedFile struct {
	Version    int
	NumRounds  int
	Encoding   string
	Ciphertext string
}

type FileVersionOptions struct {
	Version          int
	HasVersionNumber bool
	HasEncoding      bool
	HasNumRounds     bool
}

var FileVersions = map[int]FileVersionOptions{
	// just ciphertext
	1: {Version: 1, HasEncoding: false, HasNumRounds: false, HasVersionNumber: false},
	// ciphertext and encoding
	2: {Version: 2, HasEncoding: true, HasNumRounds: false, HasVersionNumber: false},
	// ciphertext, encoding, and pbkdf2 rounds
	3: {Version: 3, HasEncoding: true, HasNumRounds: true, HasVersionNumber: false},
	// ciphertext, encoding, pbkdf2 rounds, and explicit version number
	4: {Version: 4, HasEncoding: true, HasNumRounds: true, HasVersionNumber: true},
}

func FileVersion(ciphertext string) (int, error) {
	cParts := strings.Split(ciphertext, ":")
	if len(cParts) == 4 {
		return 4, nil
	}
	if len(cParts) == 3 {
		return 3, nil
	}
	if len(cParts) == 2 {
		return 2, nil
	}
	if len(cParts) == 1 {
		return 1, nil
	}
	return -1, errors.New("Invalid ciphertext")
}

func (options *FileVersionOptions) Parse(ciphertext string) (EncryptedFile, error) {
	var version string
	var encoding string
	var numRounds string
	var data string

	cParts := strings.SplitN(ciphertext, ":", 4)
	if options.Version == 4 {
		version = cParts[0]
		encoding = cParts[1]
		numRounds = cParts[2]
		data = cParts[3]
	} else if options.Version == 3 {
		encoding = cParts[0]
		numRounds = cParts[1]
		data = cParts[2]
	} else if options.Version == 2 {
		encoding = cParts[0]
		data = cParts[1]
	} else if options.Version == 1 {
		data = cParts[0]
	} else {
		return EncryptedFile{}, errors.New("Invalid version")
	}

	utils.LogDebug(fmt.Sprintf("Parsing encrypted file version %d", options.Version))

	versionSpecified := version != ""
	encodingSpecified := encoding != ""
	numRoundsSpecified := numRounds != ""

	// check required values
	if data == "" {
		return EncryptedFile{}, errors.New("Invalid format: ciphertext is required")
	}
	if options.HasVersionNumber && !versionSpecified {
		return EncryptedFile{}, errors.New("Invalid format: version is required")
	}
	if options.HasEncoding && !encodingSpecified {
		return EncryptedFile{}, errors.New("Invalid format: encoding is required")
	}
	if options.HasNumRounds && !numRoundsSpecified {
		return EncryptedFile{}, errors.New("Invalid format: number of derivation rounds is required")
	}

	file := EncryptedFile{Version: options.Version, Ciphertext: data}
	if options.HasVersionNumber && version != strconv.Itoa(options.Version) {
		return EncryptedFile{}, fmt.Errorf("Invalid file version: %s", version)
	}
	if options.HasEncoding {
		if encoding != Base64EncodingPrefix && encoding != HexEncodingPrefix {
			return EncryptedFile{}, errors.New("Invalid encoding, must be one of [base64, hex]")
		}
		file.Encoding = encoding
	} else {
		// default to hex for backwards compatibility b/c we didn't always include an encoding prefix
		// TODO remove support for v1 when releasing CLI v4 (DPLR-435)
		file.Encoding = HexEncodingPrefix
	}
	if options.HasNumRounds {
		n, err := strconv.ParseInt(numRounds, 10, 32)
		if err != nil {
			return EncryptedFile{}, errors.New("Unable to parse number of rounds")
		}
		file.NumRounds = int(n)
	} else {
		file.NumRounds = LegacyPbkdf2Rounds
	}

	return file, nil
}
