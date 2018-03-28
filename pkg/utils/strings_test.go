/*
Copyright 2018 The aerospike-controller Authors.

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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoubleQuoted(t *testing.T) {
	tests := []struct {
		provided string
		expected string
	}{
		{"test", `"test"`},
		{"", `""`},
		{"1234", `"1234"`},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, DoubleQuoted(test.provided))
	}
}

func TestHash(t *testing.T) {
	tests := []struct {
		provided string
		expected string
	}{
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"foo", "2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"},
		{`{"foo":"bar"}`, "7a38bf81f383f69433ad6e900d35b3e2385593f76a7b7ab5d4355b8ba41ee24b"},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, Hash(test.provided))
	}
}
