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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KanaryAnalysisSpec defines the desired state of KanaryAnalysis
type KanaryAnalysisSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Workload                  KanaryWorkload            `json:"workload"`
	SelectedDeployedPodMethod SelectedDeployedPodMethod `json:"selectedDeployedPodMethod"`
	Validation                KanaryValidation          `json:"validation"`
}

type SelectedDeployedPodMethod string

const (
	Ordered    SelectedDeployedPodMethod = "ordered"
	Labeled    SelectedDeployedPodMethod = "labeled"
	Disordered SelectedDeployedPodMethod = "disordered"
)

// Workload defines the analysis the workload
type KanaryWorkload struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
}

// KanaryValidation define the analysis configuration for workload
type KanaryValidation struct {
	// InitialDelay duration after the KanaryDeployment has started before validation checks is started.
	InitialDelay *metav1.Duration `json:"initialDelay,omitempty"`
	// ValidationPeriod validation checks duration.
	ValidationPeriod *metav1.Duration       `json:"validationPeriod,omitempty"`
	Interval         *metav1.Duration       `json:"interval,omitempty"`
	Items            []KanaryValidationItem `json:"items"`
}

// KanaryValidationItem define item of KanaryValidation
type KanaryValidationItem struct {
	Manual     *KanaryValidationManual     `json:"manual,omitempty"`
	LabelWatch *KanaryValidationLabelWatch `json:"labelWatch,omitempty"`
	PromQL     *KanaryValidationPromQL     `json:"promQL,omitempty"`
}

// KanaryValidationManual define workload validation status in case of manual.
type KanaryValidationManual struct {
	Status KanaryValidationManualStatus `json:"status,omitempty"`
}

type KanaryValidationManualStatus string

const (
	ValidKanaryValidationManualStatus   KanaryValidationManualStatus = "valid"
	InvalidKanaryValidationManualStatus KanaryValidationManualStatus = "invalid"
)

type KanaryValidationLabelWatch struct {
	// it means that label should be present on the workload pods.
	PodInvalidationLabels *metav1.LabelSelector `json:"podInvalidationLabels,omitempty"`
	// it means that label should be present on the workload.
	WorkloadInvalidationLabels *metav1.LabelSelector `json:"workloadInvalidationLabels,omitempty"`
}

//KanaryValidationPromQL define the promql validation configuration
type KanaryValidationPromQL struct {
	PrometheusService string        `json:"prometheusService"`
	Query             string        `json:"query"`
	ValueInRange      *ValueInRange `json:"valueInRange,omitempty"`
}

type ValueInRange struct {
	Min *float64 `json:"min"`
	Max *float64 `json:"max"`
}

// KanaryAnalysisStatus defines the observed state of KanaryAnalysis
type KanaryAnalysisStatus struct {
	// CurrentHash represents the current MD5 spec deployment template hash
	CurrentHash string `json:"currentHash,omitempty"`
	// Represents the latest available observations of a kanarydeployment's current state.
	Conditions []KanaryAnalysisCondition `json:"conditions,omitempty"`
	// Report
	Report KanaryAnalysisStatusReport `json:"report,omitempty"`
}

// KanaryDeploymentCondition describes the state of a deployment at a certain point.
type KanaryAnalysisCondition struct {
	// Type of deployment condition.
	Type KanaryAnalysisConditionType `json:"types"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

type KanaryAnalysisConditionType string

const (
	ReadyKanaryAnalysisConditionType        KanaryAnalysisConditionType = "Ready"
	DeployedKanryAnalysisConditionType      KanaryAnalysisConditionType = "Deployed"
	FailedKanaryAnalysisConditionType       KanaryAnalysisConditionType = "Failed"
	RunningKanaryAnalysisConditionType      KanaryAnalysisConditionType = "Running"
	ErroredKanaryAnalysisConditionType      KanaryAnalysisConditionType = "Errored"
	SucceedKanaryAnalysisConditionType      KanaryAnalysisConditionType = "Succeed"
	UnKnownUpdatKanaryAnalysisConditionType KanaryAnalysisConditionType = "UnKnown"

	//ScheduledKanaryAnalysisConditionType       KanaryAnalysisConditionType = "Scheduled"

	//WorkloadUpdatedKanaryAnalysisConditionType KanaryAnalysisConditionType = "WorkloadUpdated"

	//WorkloadNoUpdatKanaryAnalysisConditionType         KanaryAnalysisConditionType = "WorkloadNoUpdated"

)

type KanaryAnalysisStatusReport struct {
	Status     string `json:"status,omitempty"`
	Validation string `json:"validation,omitempty"`
	Workload   string `json:"workload,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KanaryAnalysis is the Schema for the kanaryanalyses API
// +k8s:openapi-gen=true
type KanaryAnalysis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KanaryAnalysisSpec   `json:"spec,omitempty"`
	Status KanaryAnalysisStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KanaryAnalysisList contains a list of KanaryAnalysis
type KanaryAnalysisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KanaryAnalysis `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KanaryAnalysis{}, &KanaryAnalysisList{})
	SchemeBuilder.Register(&apps.Deployment{}, &apps.DeploymentList{})

}
