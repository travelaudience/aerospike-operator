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

package pointers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBool(t *testing.T) {
	b1 := NewBool(true)
	b2 := NewBool(true)
	b3 := NewBool(false)
	assert.True(t, b1 != b2)
	assert.True(t, *b1 == *b2)
	assert.True(t, b1 != b3)
	assert.True(t, *b1 != *b3)
}

func TestNewFloat64(t *testing.T) {
	v1 := NewFloat64(1)
	v2 := NewFloat64(1.0)
	assert.True(t, v1 != v2)
	assert.True(t, *v1 == *v2)
}

func TestNewInt32(t *testing.T) {
	v1 := NewInt32(1)
	v2 := NewInt32(1)
	assert.True(t, v1 != v2)
	assert.True(t, *v1 == *v2)
}

func TestNewInt64(t *testing.T) {
	v1 := NewInt64(1)
	v2 := NewInt64(1)
	assert.True(t, v1 != v2)
	assert.True(t, *v1 == *v2)
}

func TestNewInt64FromFloat64(t *testing.T) {
	v1 := NewInt64FromFloat64(1.0)
	v2 := NewInt64FromFloat64(1.0)
	assert.True(t, v1 != v2)
	assert.True(t, *v1 == *v2)
}
