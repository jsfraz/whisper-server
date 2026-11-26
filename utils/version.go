package utils

import (
	"strconv"
	"strings"
)

// Version of this server build. Overridden at build time with
// -ldflags "-X jsfraz/shept-server/utils.Version=<tag>".
var Version = "0.1.0"

// ApiVersion is bumped when the contract changes in a way a client must notice.
const ApiVersion = "1"

// CompareVersions compares two dotted numeric versions. A suffix (a "-dev"
// build) is ignored, because only the released part decides compatibility.
//
//	@param a
//	@param b
//	@return int
func CompareVersions(a string, b string) int {
	partsA := versionParts(a)
	partsB := versionParts(b)
	for i := 0; i < 3; i++ {
		if partsA[i] != partsB[i] {
			if partsA[i] < partsB[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func versionParts(version string) [3]int {
	var parts [3]int
	core := strings.SplitN(version, "-", 2)[0]
	for i, chunk := range strings.SplitN(core, ".", 3) {
		value, err := strconv.Atoi(chunk)
		if err != nil {
			continue
		}
		parts[i] = value
	}
	return parts
}

// ClientVersionSupported reports whether a client may log in. A client that
// sends no version at all is refused: it predates the compatibility check.
//
//	@param clientVersion
//	@return bool
func ClientVersionSupported(clientVersion string) bool {
	if clientVersion == "" {
		return false
	}
	return CompareVersions(clientVersion, GetSingleton().Config.MinClientVersion) >= 0
}
