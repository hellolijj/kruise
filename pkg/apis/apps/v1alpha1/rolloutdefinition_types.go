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
)

// RolloutDefinitionSpec defines the desired state of RolloutDefinition
type RolloutDefinitionSpec struct {
	ControlResource ControlResource `json:"controlResource"`
	Path            Path            `json:"path,omitempty"`
}

// RolloutDefinitionStatus defines the observed state of RolloutDefinition
type RolloutDefinitionStatus struct {
}

// ControlResource defines the controlled resource
type ControlResource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

// Path indicates the path
type Path struct {
	SpecPath   SpecPath   `json:"specPath,omitempty"`
	StatusPath StatusPath `json:"statusPath,omitempty"`
}

// SpecPath indicates the spec path
type SpecPath struct {
	// Paused indicates the paused path of controlled workload
	Paused string `json:"paused,omitempty"`
	// Partition indicates the partition path of controlled workload.
	Partition string `json:"partition,omitempty"`
	// MaxUnavailable indicates the maxUnavailable path of controlled workload.
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
}

// StatusPath indicates the status path
type StatusPath struct {
	// ObservedGeneration indicates the observedGeneration path of controlled workload
	ObservedGeneration string `json:"observedGeneration,omitempty"`
	// Replicas indicates the replicas path of controlled workload
	Replicas string `json:"replicas"`
	// ReadyReplicas indicates the readyReplicas path of controlled workload
	ReadyReplicas string `json:"readyReplicas,omitempty"`
	// CurrentReplicas indicates the currentReplicas path of controlled workload
	CurrentReplicas string `json:"currentReplicas,omitempty"`
	// UpdatedReplicas indicates the updatedReplicas path of controlled workload
	UpdatedReplicas string `json:"updatedReplicas,omitempty"`
	// Conditions indicates the state
	Conditions string `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RolloutDefinition is the Schema for the rolloutdefinitions API
// +k8s:openapi-gen=true
type RolloutDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RolloutDefinitionSpec   `json:"spec,omitempty"`
	Status RolloutDefinitionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RolloutDefinitionList contains a list of RolloutDefinition
type RolloutDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RolloutDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RolloutDefinition{}, &RolloutDefinitionList{})
}
