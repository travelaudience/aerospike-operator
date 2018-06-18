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
