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

package flags

import (
	"flag"

	log "github.com/sirupsen/logrus"
)

// DeprecateFlags visits every flag in the specified flagset, checking if it corresponds to any of the provided name. In
// case it does, produces a warning indicating that the flag is deprecated.
func DeprecateFlags(fs *flag.FlagSet, names ...string) {
	fs.Visit(func(f *flag.Flag) {
		for _, name := range names {
			if f.Name == name {
				log.Warnf("the --%s flag is deprecated and will be removed in a future release", name)
			}
		}
	})
}
