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

package backuphandler

import (
	"fmt"
	"time"

	"k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
)

func (h *AerospikeBackupsHandler) waitForJobCondition(job *v1.Job, timeout time.Duration, fn watch.ConditionFunc) error {
	w, err := h.kubeclientset.BatchV1().Jobs(job.Namespace).Watch(listoptions.ObjectByName(job.Name))
	if err != nil {
		return err
	}
	start := time.Now()
	last, err := watch.Until(timeout, w, fn)
	if err != nil {
		if err == watch.ErrWatchClosed {
			if t := timeout - time.Since(start); t > 0 {
				return h.waitForJobCondition(job, t, fn)
			}
		}
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for job %s", meta.Key(job))
	}
	return nil
}
