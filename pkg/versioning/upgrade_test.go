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

func TestIsMajor(t *testing.T) {
	tests := []struct {
		upgrade VersionUpgrade
		result  bool
	}{
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 0, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{5, 1, 1, 1},
		}, true},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 1, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 1, 0, 0},
			Version{4, 1, 1, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 0, 0, 1},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{3, 0, 0, 0},
		}, false},
	}
	for _, test := range tests {
		assert.Equal(t, test.upgrade.isMajorUpgrade(), test.result)
	}
}

func TestIsMinor(t *testing.T) {
	tests := []struct {
		upgrade VersionUpgrade
		result  bool
	}{
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 0, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{5, 1, 1, 1},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 1, 0, 0},
		}, true},
		{VersionUpgrade{
			Version{4, 1, 0, 0},
			Version{4, 1, 1, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 0, 0, 1},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{3, 0, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{3, 1, 0, 0},
		}, false},
	}
	for _, test := range tests {
		assert.Equal(t, test.upgrade.isMinorUpgrade(), test.result)
	}
}

func TestIsPatch(t *testing.T) {
	tests := []struct {
		upgrade VersionUpgrade
		result  bool
	}{
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 0, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{5, 1, 1, 1},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 1, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 1, 0, 0},
			Version{4, 1, 1, 0},
		}, true},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 0, 0, 1},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{3, 0, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 3, 0, 0},
			Version{3, 2, 1, 0},
		}, false},
	}
	for _, test := range tests {
		assert.Equal(t, test.upgrade.isPatchUpgrade(), test.result)
	}
}

func TestIsRevision(t *testing.T) {
	tests := []struct {
		upgrade VersionUpgrade
		result  bool
	}{
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 0, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{5, 1, 1, 1},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 1, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 1, 0, 0},
			Version{4, 1, 1, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{4, 0, 0, 1},
		}, true},
		{VersionUpgrade{
			Version{4, 0, 0, 0},
			Version{3, 0, 0, 0},
		}, false},
		{VersionUpgrade{
			Version{4, 3, 2, 0},
			Version{3, 2, 1, 1},
		}, false},
	}
	for _, test := range tests {
		assert.Equal(t, test.upgrade.isRevisionUpgrade(), test.result)
	}
}
