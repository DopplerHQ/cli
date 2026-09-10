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
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// ProgramVersion the current version of this program
var ProgramVersion = "dev"

// ProgramName is the invoked command name (cobra Use / help text) and the identity the
// update self-management keys on. ConfigDirName / ConfigFileName locate the on-disk
// config under the user's home. All three are build-time-injectable so a renamed
// distribution — e.g. the agent-proxy demo build — can flip its name and config location
// without forking the code:
//
//	-ldflags "-X github.com/DopplerHQ/cli/pkg/version.ProgramName=doppler-agent \
//	          -X github.com/DopplerHQ/cli/pkg/version.ConfigDirName=.doppler-agent \
//	          -X github.com/DopplerHQ/cli/pkg/version.ConfigFileName=.doppler-agent.yaml"
//
// The point is a rebranded build never collides with a customer's production `doppler`:
// a distinct name on PATH, and a separate config dir so it can't read or clobber their
// real credentials.
var (
	ProgramName    = "doppler"
	ConfigDirName  = ".doppler"
	ConfigFileName = ".doppler.yaml"
)

// IsRenamed reports whether this is a rebranded distribution rather than the official
// doppler CLI. A renamed build turns off update self-management (the `update` command and
// the startup check) — it must never fetch and overwrite itself with the production binary.
func IsRenamed() bool {
	return ProgramName != "doppler"
}

// Version semver
type Version struct {
	Major int16
	Minor int16
	Patch int16
}

// Unwrap get the original error
func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsDevelopment whether the CLI is running in development mode (not a released version)
func IsDevelopment() bool {
	return strings.HasPrefix(ProgramVersion, "dev")
}

// CompareVersions returns -1 if first is greater, 1 if second is greater, and 0 otherwise
func CompareVersions(a Version, b Version) int {
	// major version
	if a.Major > b.Major {
		return -1
	}
	if b.Major > a.Major {
		return 1
	}

	// minor version
	if a.Minor > b.Minor {
		return -1
	}
	if b.Minor > a.Minor {
		return 1
	}

	// patch version
	if a.Patch > b.Patch {
		return -1
	}
	if b.Patch > a.Patch {
		return 1
	}

	return 0
}

// ParseVersion from a string
func ParseVersion(s string) (Version, error) {
	if strings.HasPrefix(s, "v") {
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("Invalid version %s", s)
	}

	var v Version
	var major int64
	var minor int64
	var patch int64
	var err error
	if major, err = strconv.ParseInt(parts[0], 10, 16); err != nil {
		return Version{}, fmt.Errorf("Invalid version %s", s)
	}
	if minor, err = strconv.ParseInt(parts[1], 10, 16); err != nil {
		return Version{}, fmt.Errorf("Invalid version %s", s)
	}
	if patch, err = strconv.ParseInt(parts[2], 10, 16); err != nil {
		return Version{}, fmt.Errorf("Invalid version %s", s)
	}

	v.Major = int16(major)
	v.Minor = int16(minor)
	v.Patch = int16(patch)
	return v, nil
}

// Normalize prepends a 'v' to a version (e.g. 1.0.0 -> v1.0.0)
func Normalize(version string) string {
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}

	return version
}
