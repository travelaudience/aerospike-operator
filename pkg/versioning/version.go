package versioning

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a version of Aerospike.
type Version struct {
	Major    int
	Minor    int
	Patch    int
	Revision int
}

// NewVersionFromString parses the specified version string into the
// corresponding Version struct.
func NewVersionFromString(versionString string) (Version, error) {
	// split versionString by "."
	s := strings.Split(versionString, ".")
	if len(s) < 3 || len(s) > 4 {
		return Version{}, fmt.Errorf("invalid version scheme")
	}
	// parse each of the parts as an integer
	parts := make([]int, 4)
	for i, s := range s {
		d, err := strconv.Atoi(s)
		if err != nil {
			return Version{}, err
		}
		parts[i] = d
	}
	// return the populated struct
	return Version{parts[0], parts[1], parts[2], parts[3]}, nil
}

// String returns the string representation of the current struct.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Revision)
}

// IsSupported indicated whether the version of Aerospike represented by the
// current struct is supported by the operator.
func (v Version) IsSupported() bool {
	return contains(AerospikeServerSupportedVersions, v.String())
}

// contains returns a boolean indicating whether e is contained in the s slice.
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
