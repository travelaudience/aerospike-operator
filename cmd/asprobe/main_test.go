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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStatistics(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{
			input:    "",
			expected: map[string]string{},
		},
		{
			input: "foo=bar",
			expected: map[string]string{
				"foo": "bar",
			},
		},
		{
			input: "foo=;bar=baz;",
			expected: map[string]string{
				"foo": "",
				"bar": "baz",
			},
		},
		{
			input: "foo=bar;bar=baz;",
			expected: map[string]string{
				"foo": "bar",
				"bar": "baz",
			},
		},
		{
			input: "foo=bar;bar=baz;qux=1",
			expected: map[string]string{
				"foo": "bar",
				"bar": "baz",
				"qux": "1",
			},
		},
		{
			input: " foo =  bar; bar= baz  ; qux  =1; ",
			expected: map[string]string{
				"foo": "bar",
				"bar": "baz",
				"qux": "1",
			},
		},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, parseStatistics(test.input))
	}
}
