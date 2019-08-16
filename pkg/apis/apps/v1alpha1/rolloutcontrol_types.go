/*
Copyright 2019 The Kruise Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// RolloutControlSpec defines the desired state of RolloutControl
type RolloutControlSpec struct {
	Resource        CompleteResource `json:"resource"`
	RolloutStrategy RolloutStrategy  `json:"rolloutstrategy,omitempty"`
}

type CompleteResource struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	NameSpace  string `json:"namespace"`
	Name       string `json:"name"`
}

type RolloutStrategy struct {
	// Partition indicates the ordinal at which the workload should be
	// partitioned.
	// Default value is 0.
	// +optional
	Partition int32 `json:"partition,omitempty"`
	// The maximum number of pods that can be unavailable during the update.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// Absolute number is calculated from percentage by rounding down.
	// Also, maxUnavailable can just be allowed to work with Parallel podManagementPolicy.
	// Defaults to 1.
	// +optional
	MaxUnavailable intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// Paused indicates that the process is paused.
	// Default value is false
	// +optional
	Paused bool `json:"paused,omitempty"`
}

// RolloutControlStatus defines the observed state of RolloutControl
type RolloutControlStatus struct {
	// ObservedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Replicas is the number of Pods created by the CR controller.
	Replicas int32 `json:"replicas"`

	// ReadyReplicas is the number of Pods created by the CR controller that have a Ready Condition.
	ReadyReplicas int32 `json:"readyReplicas"`

	// CurrentReplicas is the number of Pods created by the CR controller from the CR version
	// indicated by currentRevision.
	CurrentReplicas int32 `json:"currentReplicas"`

	// UpdatedReplicas is the number of Pods created by the CR controller from the CR version
	// indicated by updateRevision.
	UpdatedReplicas int32 `json:"updatedReplicas"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RolloutControl is the Schema for the rolloutcontrols API
// +k8s:openapi-gen=true
type RolloutControl struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RolloutControlSpec   `json:"spec,omitempty"`
	Status RolloutControlStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RolloutControlList contains a list of RolloutControl
type RolloutControlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RolloutControl `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RolloutControl{}, &RolloutControlList{})
}
