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

package framework

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// conditionFunc represents a function that returns a boolean indicating whether a given condition is true and an error
// if the evaluation of such condition was not successful.
type conditionFunc func() (bool, error)

// retry retries evaluating the specified function every d, until fn evaluates to true, returns an error or max is
// exceeded.
func retry(d time.Duration, max int, fn conditionFunc) error {
	tick := time.NewTicker(d)
	defer tick.Stop()

	for i := 0; i < max; i++ {
		if res, err := fn(); err != nil {
			log.Warnf("failed to evaluate condition: %v", err)
		} else if res {
			return nil
		}
		<-tick.C
	}

	return fmt.Errorf("the maximum number of retries has been exceeded")
}
