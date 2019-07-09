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
	"k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExecAction describes a "run in container" action.
type ExecAction struct {
	// Command is the command line to execute inside the container, the working directory for the
	// command  is root ('/') in the container's filesystem. The command is simply exec'd, it is
	// not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use
	// a shell, you need to explicitly call out to that shell.
	// Exit status of 0 is treated as live/healthy and non-zero is unhealthy.
	// +optional
	Command   []string `json:"command,omitempty" protobuf:"bytes,1,rep,name=command"`
	Container string   `json:"container" protobuf:"bytes,1,opt,name=container"`
}

// Handler defines a specific action that should be taken
type Handler struct {
	// One and only one of the following should be specified.
	// Exec specifies the action to take.
	// +optional
	Exec *ExecAction `json:"exec,omitempty" protobuf:"bytes,1,opt,name=exec"`
	// HTTPGet specifies the http request to perform.
	// +optional
	HTTPGet *v1.HTTPGetAction `json:"httpGet,omitempty" protobuf:"bytes,2,opt,name=httpGet"`
}

// MarkProbe describes a probe check to be performed against a pod to determine whether it is label
type MarkProbe struct {
	// The action taken to determine the health of a container
	Handler `json:",inline" protobuf:"bytes,1,opt,name=handler"`
	// Number of seconds after which the probe times out.
	// Defaults to 1 second. Minimum value is 1.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,3,opt,name=timeoutSeconds"`
	// How often (in seconds) to perform the probe.
	// Default to 10 seconds. Minimum value is 1.
	// +optional
	PeriodSeconds int32 `json:"periodSeconds,omitempty" protobuf:"varint,4,opt,name=periodSeconds"`
}

// PodProbeMarkerSpec defines the desired state of PodProbeMarker
type PodProbeMarkerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Selector    *metav1.LabelSelector `json:"selector"`
	Labels      map[string]string     `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	MarkerProbe MarkProbe             `json:"markProbe"`
}

// PodProbeMarkerStatus defines the observed state of PodProbeMarker
type PodProbeMarkerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// replicas is the number of Pods mark by the MarkerProbeSet
	Replicas int32 `json:"replicas"`
	// readyReplicas is the number of Pods created by the StatefulSet controller that have a Ready Condition.
	SelectedReplicas int32  `json:"selectedReplicas"`
	MarkPodName      string `json:"markPodName"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodProbeMarker is the Schema for the podprobemarkers API
// +k8s:openapi-gen=true
type PodProbeMarker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodProbeMarkerSpec   `json:"spec,omitempty"`
	Status PodProbeMarkerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodProbeMarkerList contains a list of PodProbeMarker
type PodProbeMarkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodProbeMarker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodProbeMarker{}, &PodProbeMarkerList{})
}
