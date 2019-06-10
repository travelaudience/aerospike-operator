/*
Copyright 2018 The aerospike-operator Authors.

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
