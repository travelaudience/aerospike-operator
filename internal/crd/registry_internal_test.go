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

package crd

import (
	"sync"
	"testing"
	"time"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"

	"github.com/stretchr/testify/assert"
	extsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	kubetesting "k8s.io/client-go/testing"
)

func TestCreateCRDFailsOnInternalError(t *testing.T) {
	extsClient := fake.NewSimpleClientset()
	extsClient.PrependReactor("create", "customresourcedefinitions", func(_ kubetesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewInternalError(assert.AnError)
	})
	r := NewCRDRegistry(extsClient)
	err := r.createCRD(crds[0])
	assert.Error(t, err)
}

func TestCreateCRDDoesNotFailOnAlreadyExists(t *testing.T) {
	extsClient := fake.NewSimpleClientset()
	extsClient.PrependReactor("create", "customresourcedefinitions", func(_ kubetesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewAlreadyExists(schema.GroupResource{}, "")
	})
	r := NewCRDRegistry(extsClient)
	err := r.createCRD(crds[0])
	assert.NoError(t, err)
}

func TestAwaitCRDWaitsForEstablishedCondition(t *testing.T) {
	extsClient := fake.NewSimpleClientset()
	fw := watch.NewFake()
	extsClient.PrependWatchReactor("customresourcedefinitions", func(_ kubetesting.Action) (bool, watch.Interface, error) {
		return true, fw, nil
	})
	r := NewCRDRegistry(extsClient)

	var (
		wg  sync.WaitGroup
		err error
		t0  time.Time
		t1  time.Time
		dt  = 3 * time.Second
	)

	// await for the CRD to be established in a goroutine so we can send watch events from this one
	t0 = time.Now()
	go func() {
		defer wg.Done()
		err = r.awaitCRD(crds[0])
		t1 = time.Now()
	}()
	wg.Add(1)

	// wait for dt time so we can assert that awaitCRD indeed waits for the update event
	<-time.After(dt)
	fw.Modify(establishedCRD(crds[0]))
	wg.Wait()
	assert.True(t, t1.Sub(t0) >= dt)
	assert.NoError(t, err)
}

func establishedCRD(crd *extsv1beta1.CustomResourceDefinition) *extsv1beta1.CustomResourceDefinition {
	res := crd.DeepCopy()
	res.Status.Conditions = append(crd.Status.Conditions, extsv1beta1.CustomResourceDefinitionCondition{
		Type:   extsv1beta1.Established,
		Status: extsv1beta1.ConditionTrue,
	})
	return res
}
