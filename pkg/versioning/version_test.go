package versioning

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVersionFromString(t *testing.T) {
	tests := []struct {
		versionString string
		expectError   bool
	}{
		{"4.0", true},
		{"4.0.0.0", false},
		{"1.2.3.4.5", true},
		{"4.1.0", false},
		{"4.1.0.1", false},
	}
	for _, test := range tests {
		_, err := NewVersionFromString(test.versionString)
		if test.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		version       Version
		versionString string
	}{
		{Version{4, 0, 0, 4}, "4.0.0.4"},
		{Version{1, 2, 3, 4}, "1.2.3.4"},
	}
	for _, test := range tests {
		assert.Equal(t, test.version.String(), test.versionString)
	}
}
