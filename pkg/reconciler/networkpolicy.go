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

package reconciler

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

var (
	protocolTCP = v1.ProtocolTCP
	protocolUDP = v1.ProtocolUDP
)

func (r *AerospikeClusterReconciler) ensureNetworkPolicy(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	policy := networkv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: aerospikeCluster.Name,
			Labels: map[string]string{
				selectors.LabelAppKey:     selectors.LabelAppVal,
				selectors.LabelClusterKey: aerospikeCluster.Name,
			},
			Namespace: aerospikeCluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         aerospikev1alpha1.SchemeGroupVersion.String(),
					Kind:               kind,
					Name:               aerospikeCluster.Name,
					UID:                aerospikeCluster.UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
		},
		Spec: networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					selectors.LabelAppKey:     selectors.LabelAppVal,
					selectors.LabelClusterKey: aerospikeCluster.Name,
				},
			},
			PolicyTypes: []networkv1.PolicyType{
				networkv1.PolicyTypeIngress,
				networkv1.PolicyTypeEgress,
			},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					From: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									selectors.LabelAppKey:     selectors.LabelAppVal,
									selectors.LabelClusterKey: aerospikeCluster.Name,
								},
							},
						},
					},
					Ports: []networkv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: fabricPort,
							},
						},
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: heartbeatPort,
							},
						},
					},
				},
				{
					Ports: []networkv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: servicePort,
							},
						},
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: infoPort,
							},
						},
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: asprobePort,
							},
						},
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: aspromPort,
							},
						},
					},
				},
			},
			Egress: []networkv1.NetworkPolicyEgressRule{
				{
					To: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									selectors.LabelAppKey:     selectors.LabelAppVal,
									selectors.LabelClusterKey: aerospikeCluster.Name,
								},
							},
						},
					},
					Ports: []networkv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: fabricPort,
							},
						},
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: heartbeatPort,
							},
						},
					},
				},
				{
					Ports: []networkv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port: &intstr.IntOrString{
								IntVal: 53,
							},
						},
						{
							Protocol: &protocolUDP,
							Port: &intstr.IntOrString{
								IntVal: 53,
							},
						},
					},
				},
			},
		},
	}

	if _, err := r.kubeclientset.NetworkingV1().NetworkPolicies(aerospikeCluster.Namespace).Create(&policy); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		}).Debug("networkpolicy already exists")
		return nil
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debug("networkpolicy created")
	return nil
}
