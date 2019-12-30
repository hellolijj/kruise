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

package podprobemarker

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/pkg/probe"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

var (
	testScheme *runtime.Scheme
)

func init() {
	testScheme = runtime.NewScheme()
	_ = appsv1alpha1.AddToScheme(testScheme)
	_ = corev1.AddToScheme(testScheme)
}

func TestReconcile(t *testing.T) {
	podPorbeMarker := createHttpExecPodProbeMarker("test-http-probe-marker", map[string]string{"run": "nginx"}, "", 80)
	pods := make([]runtime.Object, 10)
	for i := 0; i < 10; i++ {
		if i%3 == 0 {
			pods[i] = createPod("pod-"+fmt.Sprintf("%d", i), map[string]string{"run": "nginx"})
			continue
		}
		pods[i] = createPod("pod-"+fmt.Sprintf("%d", i), map[string]string{"run": "other"})
	}
	runtimeObj := []runtime.Object{podPorbeMarker}
	runtimeObj = append(runtimeObj, pods...)
	r := createReconcile(testScheme, runtimeObj...)

	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: podPorbeMarker.Namespace,
			Name:      podPorbeMarker.Name,
		},
	}
	_, err := r.Reconcile(request)
	assert.NoError(t, err)

	latestPodProbeMarker, err := findLatestPodProbeMarker(r.Client, podPorbeMarker)
	assert.NoError(t, err)
	assert.Equal(t, int32(10), latestPodProbeMarker.Status.Matched)
	assert.Equal(t, int32(0), latestPodProbeMarker.Status.Succeeded)
	assert.Equal(t, int32(10), latestPodProbeMarker.Status.Failed)
}

// test http probe
func TestHttpProbe(t *testing.T) {
	podPorbeMarker := createHttpExecPodProbeMarker("test-http-probe-marker", nil, "", 80)
	pod := createPod("test", nil)
	r := createReconcile(testScheme, podPorbeMarker, pod)
	res, err := r.httpProbe(pod, &podPorbeMarker.Spec.MarkerProbe)
	assert.NoError(t, err)
	assert.Equal(t, probe.Failure, res) // because there is no server to unit test.
}

// test exec probe
//func TestExecProbe(t *testing.T) {
//	podPorbeMarker := createExecPodProbeMarker("podProbeMarkerInstance", nil, []string{"ls", "."}, "nginx")
//	pod := createPod("test", nil)
//	r := createReconcile(testScheme, podPorbeMarker, pod)
//	res, err := r.execProbe(pod, &podPorbeMarker.Spec.MarkerProbe)
//	assert.NoError(t, err)
//	assert.Equal(t, probe.Success, res)
//}

// test probe status
func TestSyncStatus(t *testing.T) {
	podPorbeMarker := createExecPodProbeMarker("podProbeMarkerInstance", nil, nil, "")
	status := &appsv1alpha1.PodProbeMarkerStatus{
		Matched:   10,
		Succeeded: 4,
		Failed:    6,
	}
	r := createReconcile(testScheme, podPorbeMarker)
	err := r.syncStatus(podPorbeMarker, status)
	assert.NoError(t, err)
	latestPodProbeMarker, err := findLatestPodProbeMarker(r.Client, podPorbeMarker)
	assert.NoError(t, err)
	assert.Equal(t, int32(10), latestPodProbeMarker.Status.Matched)
	assert.Equal(t, int32(4), latestPodProbeMarker.Status.Succeeded)
	assert.Equal(t, int32(6), latestPodProbeMarker.Status.Failed)
}

// test getContainerFromMarkProbe
func TestFindContainerFromMarkProbe(t *testing.T) {
	pod := createPod("test", map[string]string{"run": "hello-world"})
	podPorbeMarker := createExecPodProbeMarker("podProbeMarkerInstance", map[string]string{"run": "nginx"}, []string{"ls", "/"}, "nginx")
	c := findContainerFromMarkProbe(pod, &podPorbeMarker.Spec.MarkerProbe)
	assert.Equal(t, "nginx", c.Name)
}

func createReconcile(scheme *runtime.Scheme, initObjs ...runtime.Object) ReconcilePodProbeMarker {
	fakeClient := fake.NewFakeClientWithScheme(scheme, initObjs...)
	reconcileRes := ReconcilePodProbeMarker{
		Client: fakeClient,
		scheme: scheme,
	}
	return reconcileRes
}

func createPod(name string, label map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: corev1.NamespaceDefault,
			Labels:    label,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:1.15.1",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}

func createExecPodProbeMarker(name string, matchLabels map[string]string, command []string, container string) *appsv1alpha1.PodProbeMarker {
	selector := metav1.LabelSelector{
		MatchLabels: matchLabels,
	}
	labels := map[string]string{"runExec": "success"}
	httpProbe := appsv1alpha1.MarkProbe{
		Handler: appsv1alpha1.Handler{
			Exec: &appsv1alpha1.ExecAction{
				Command:   command,
				Container: container,
			},
		},
		TimeoutSeconds: 4,
		PeriodSeconds:  5,
	}
	return createPodProbeMarker(name, &selector, labels, httpProbe)
}

func createHttpExecPodProbeMarker(name string, matchLabels map[string]string, path string, port int32) *appsv1alpha1.PodProbeMarker {
	selector := metav1.LabelSelector{
		MatchLabels: matchLabels,
	}
	labels := map[string]string{"runHttp": "success"}
	httpProbe := appsv1alpha1.MarkProbe{
		Handler: appsv1alpha1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:        path,
				Port:        intstr.IntOrString{Type: intstr.Int, IntVal: port},
				Host:        "localhost",
				Scheme:      "http",
				HTTPHeaders: nil,
			},
		},
		TimeoutSeconds: 4,
		PeriodSeconds:  5,
	}
	return createPodProbeMarker(name, &selector, labels, httpProbe)
}

func createPodProbeMarker(name string, selector *metav1.LabelSelector, labels map[string]string, probe appsv1alpha1.MarkProbe) *appsv1alpha1.PodProbeMarker {
	return &appsv1alpha1.PodProbeMarker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: corev1.NamespaceDefault,
		},
		Spec: appsv1alpha1.PodProbeMarkerSpec{
			Selector:    selector,
			Labels:      labels,
			MarkerProbe: probe,
		},
		Status: appsv1alpha1.PodProbeMarkerStatus{},
	}
}

func findLatestPodProbeMarker(c client.Client, podProbeMarker *appsv1alpha1.PodProbeMarker) (*appsv1alpha1.PodProbeMarker, error) {
	newPodProbeMarker := &appsv1alpha1.PodProbeMarker{}
	keyPodProbeMarker := types.NamespacedName{
		Namespace: podProbeMarker.Namespace,
		Name:      podProbeMarker.Name,
	}
	if err := c.Get(context.TODO(), keyPodProbeMarker, newPodProbeMarker); err != nil {
		return nil, err
	}
	return newPodProbeMarker, nil
}
