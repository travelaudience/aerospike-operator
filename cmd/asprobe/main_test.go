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
