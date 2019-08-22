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
	"reflect"
	"sync"

	"k8s.io/klog"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
)

// ResourcePathTable is used to map resource to path
var ResourcePathTable ResourcePath

func init() {
	ResourcePathTable = ResourcePath{
		pathTable: make(map[appsv1alpha1.ControlResource]*appsv1alpha1.Path),
	}
}

// ResourcePath defines a map struct of resource and path
type ResourcePath struct {
	mutex     sync.RWMutex
	pathTable map[appsv1alpha1.ControlResource]*appsv1alpha1.Path
}

func (rp *ResourcePath) Get(apiVersion, resource string) (result *appsv1alpha1.Path) {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()
	key := appsv1alpha1.ControlResource{APIVersion: apiVersion, Resource: resource}
	result, ok := rp.pathTable[key]
	if !ok {
		klog.Infof("failed to get ResourcePath of the key %v", key)
		return nil
	}
	return result
}

func (rp *ResourcePath) Set(key appsv1alpha1.ControlResource, path *appsv1alpha1.Path) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()
	if p, ok := rp.pathTable[key]; ok {
		// The path was already stored.
		if reflect.DeepEqual(p, path) {
			klog.Infof("DeepEqual of the key %v", key)
			return
		}
		rp.pathTable[key] = path
		klog.Infof("succeed to update ResourcePath of the key %v", key)
	} else {
		rp.pathTable[key] = path
		klog.Infof("succeed to set ResourcePath of the key %v", key)
	}
}
