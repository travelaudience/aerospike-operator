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

package asutils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	as "github.com/aerospike/aerospike-client-go"
)

const timeout = 10 * time.Second

func GetClusterSize(host string, port int) (int, error) {
	c, err := as.NewConnection(fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return 0, err
	}
	r, err := as.RequestInfo(c, "statistics")
	if err != nil {
		return 0, err
	}
	if str, ok := ParseStatistics(r["statistics"])["cluster_size"]; !ok {
		return 0, fmt.Errorf("cluster_size is not present")
	} else {
		return strconv.Atoi(str)
	}
}

// ParseStatistics parses a string in the form a=b;c=d; into a map[string]string, trimming whitespace in the process.
func ParseStatistics(stats string) map[string]string {
	res := make(map[string]string)
	pairs := strings.Split(stats, ";")
	for _, pair := range pairs {
		r := strings.Split(pair, "=")
		if len(r) == 2 {
			res[strings.TrimSpace(r[0])] = strings.TrimSpace(r[1])
		}
	}
	return res
}
