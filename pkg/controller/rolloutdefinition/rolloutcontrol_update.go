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

package rolloutdefinition

import (
	"strings"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	kuriseclient "github.com/openkruise/kruise/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

var ResourceControlTable map[appsv1alpha1.ControlResource][]types.NamespacedName

func init() {
	ResourceControlTable = make(map[appsv1alpha1.ControlResource][]types.NamespacedName)
}

func UpdateStatusFromResource(controlResource appsv1alpha1.ControlResource, resource map[string]interface{}) error {
	resourcePath := ResourcePathTable.Get(controlResource.APIVersion, controlResource.Resource)
	if resourcePath == nil {
		klog.Info("have no resourcePath")
		return nil
	}
	rollourControls := ResourceControlTable[controlResource]
	klog.Infof("qwkLog：UpdateStatusFromResource resource.Object : %v", resource)

	// update all rolloutControl
	for i := 0; i < len(rollourControls); i++ {
		klog.Infof("qwkLog：begin UpdateStatusFromResource rollourControl : %v", rollourControls[i])

		rolloutCtl, err := kuriseclient.GetGenericClient().KruiseClient.AppsV1alpha1().RolloutControls(rollourControls[i].Namespace).Get(rollourControls[i].Name, metav1.GetOptions{})
		if err != nil {
			klog.Info("qwkLog：failed to get rolloutControl")
			return err
		}
		updateRC := rolloutCtl.DeepCopy()

		// set replicas
		if resourcePath.StatusPath.Replicas != "" {
			replicasPathArr := strings.Split(resourcePath.StatusPath.Replicas, ".")
			klog.Infof("qwkLog：replicasPathArr : %v", replicasPathArr)
			replicasV, b, err := unstructured.NestedFieldNoCopy(resource, replicasPathArr...)
			if err != nil {
				return err
			}
			if b == true {
				klog.Infof("qwkLog：get replicas value: %v", replicasV)
				updateRC.Status.Replicas = int32(replicasV.(int64))
				klog.Infof("qwkLog：end set updateRC replicas value: %v", updateRC.Status.Replicas)
			} else {
				klog.Info("can't get path field of replicas")
			}
		} else {
			klog.Info("replicas is not supported")
		}
		// set readyReplicas
		if resourcePath.StatusPath.ReadyReplicas != "" {
			readyReplicasPathArr := strings.Split(resourcePath.StatusPath.ReadyReplicas, ".")
			readyReplicasV, b, err := unstructured.NestedFieldNoCopy(resource, readyReplicasPathArr...)
			if err != nil {
				return err
			}
			if b == true {
				klog.Infof("qwkLog：get readyReplicas value: %v", readyReplicasV)
				updateRC.Status.ReadyReplicas = int32(readyReplicasV.(int64))
				klog.Infof("qwkLog：end set updateRC readyReplicas value: %v", updateRC.Status.ReadyReplicas)
			} else {
				klog.Info("can't get path field of readyReplicas")
			}
		} else {
			klog.Info("readyReplicas is not supported")
		}
		// set currentReplicas
		if resourcePath.StatusPath.CurrentReplicas != "" {
			currentReplicasPathArr := strings.Split(resourcePath.StatusPath.CurrentReplicas, ".")
			currentReplicasV, b, err := unstructured.NestedFieldNoCopy(resource, currentReplicasPathArr...)
			if err != nil {
				return err
			}
			if b == true {
				klog.Infof("qwkLog：get currentReplicas value: %v", currentReplicasV)
				updateRC.Status.CurrentReplicas = int32(currentReplicasV.(int64))
				klog.Infof("qwkLog：end set updateRC currentReplicas value: %v", updateRC.Status.CurrentReplicas)
			} else {
				klog.Info("can't get path field of currentReplicas")
			}
		} else {
			klog.Info("currentReplicas is not supported")
		}
		// set updatedReplicas
		if resourcePath.StatusPath.UpdatedReplicas != "" {
			updatedReplicasPathArr := strings.Split(resourcePath.StatusPath.UpdatedReplicas, ".")
			updatedReplicasV, b, err := unstructured.NestedFieldNoCopy(resource, updatedReplicasPathArr...)
			if err != nil {
				return err
			}
			if b == true {
				klog.Infof("qwkLog：get updatedReplicas value: %v", updatedReplicasV)
				updateRC.Status.UpdatedReplicas = int32(updatedReplicasV.(int64))
				klog.Infof("qwkLog：end set updateRC updatedReplicas value: %v", updateRC.Status.UpdatedReplicas)
			} else {
				klog.Info("can't get path field of updatedReplicas")
			}
		} else {
			klog.Info("updatedReplicas is not supported")
		}

		// set observedGeneration
		if resourcePath.StatusPath.ObservedGeneration != "" {
			observedGenerationPathArr := strings.Split(resourcePath.StatusPath.ObservedGeneration, ".")
			observedGenerationV, b, err := unstructured.NestedFieldNoCopy(resource, observedGenerationPathArr...)
			if err != nil {
				return err
			}
			if b == true {
				klog.Infof("qwkLog：get observedGeneration value: %v", observedGenerationV)
				updateRC.Status.ObservedGeneration = observedGenerationV.(int64)
				klog.Infof("qwkLog：end set updateRC observedGeneration value: %v", updateRC.Status.ObservedGeneration)
			} else {
				klog.Info("can't get path field of observedGeneration")
			}
		} else {
			klog.Info("observedGeneration is not supported")
		}

		klog.Infof("qwkLog：begin update rolloutControl")
		updated, err := kuriseclient.GetGenericClient().KruiseClient.AppsV1alpha1().RolloutControls(rollourControls[i].Namespace).Update(updateRC)
		if err != nil {
			klog.Info("qwkLog：failed to update rolloutControl")
			return err
		}
		klog.Infof("qwkLog：end update rolloutControl : %v", updated)
	}

	return nil
}
