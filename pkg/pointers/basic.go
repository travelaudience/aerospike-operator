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

package pointers

// NewBool returns a pointer to a bool holding the value of v.
func NewBool(v bool) *bool {
	res := v
	return &res
}

// NewFloat64 returns a pointer to a float64 holding the value of v.
func NewFloat64(v float64) *float64 {
	res := float64(v)
	return &res
}

// NewInt32 returns a pointer to a int32 holding the value of v.
func NewInt32(v int32) *int32 {
	res := int32(v)
	return &res
}

// NewInt64 returns a pointer to a int64 holding the value of v.
func NewInt64(v int64) *int64 {
	res := int64(v)
	return &res
}

// NewInt64FromFloat64 returns a pointer to an int64 holding the value of v
// after conversion to int64.
func NewInt64FromFloat64(v float64) *int64 {
	res := int64(v)
	return &res
}
