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

package time

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration extends the time.ParseDuration to accept "d" (days)
// as a suffix.
func ParseDuration(s string) (time.Duration, error) {
	if strings.Contains(s, "d") {
		if v, err := strconv.ParseFloat(strings.TrimSuffix(s, "d"), 64); err == nil {
			return time.ParseDuration(fmt.Sprintf("%fh", v*24))
		}
		x := strings.Replace(s, "d", "h", 1)
		d, err := time.ParseDuration(x)
		err = fmt.Errorf(strings.Replace(err.Error(), x, s, -1))
		return d, err
	}
	return time.ParseDuration(s)
}
