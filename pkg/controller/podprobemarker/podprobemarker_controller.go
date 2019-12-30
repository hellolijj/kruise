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
	"bytes"
	"context"
	"fmt"
	"time"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	genericclient "github.com/openkruise/kruise/pkg/client"
	"github.com/openkruise/kruise/pkg/util/gate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	schemeutil "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/probe"
	"k8s.io/kubernetes/pkg/probe/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new PodProbeMarker Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	if !gate.ResourceEnabled(&appsv1alpha1.PodProbeMarker{}) {
		return nil
	}
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePodProbeMarker{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("podprobemarker-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to PodProbeMarker
	err = c.Watch(&source.Kind{Type: &appsv1alpha1.PodProbeMarker{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePodProbeMarker{}

// ReconcilePodProbeMarker reconciles a PodProbeMarker object
type ReconcilePodProbeMarker struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PodProbeMarker object and makes changes based on the state read
// and what is in the PodProbeMarker.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps.kruise.io,resources=podprobemarkers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kruise.io,resources=podprobemarkers/status,verbs=get;update;patch
func (r *ReconcilePodProbeMarker) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	podProbeMarker := &appsv1alpha1.PodProbeMarker{}
	err := r.Get(context.TODO(), request.NamespacedName, podProbeMarker)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	pods, err := r.listPodFromProbeMarker(podProbeMarker)
	if err != nil {
		return reconcile.Result{}, nil
	}

	var status appsv1alpha1.PodProbeMarkerStatus
	status.Matched = int32(len(pods))
	for _, p := range pods {
		klog.V(4).Infof("find pod %v, to probe", p.Name)

		probeResult, err := r.probe(podProbeMarker, p)
		if err != nil || probeResult == probe.Failure {
			status.Failed++
			klog.Errorf("exec pod %s err %v", p.Name, err)
			continue
		}

		if probeResult == probe.Success {
			newPod := updatePodLabels(p, podProbeMarker.Spec.Labels)
			if err := r.Update(context.TODO(), newPod); err != nil {
				status.Failed++
				klog.Errorf("update pod %s err %v", p.Name, err)
				continue
			}
			status.Succeeded++
		}
	}

	period := podProbeMarker.Spec.MarkerProbe.PeriodSeconds
	if period != 0 {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Duration(period) * time.Second,
		}, r.syncStatus(podProbeMarker, &status)
	}

	return reconcile.Result{}, r.syncStatus(podProbeMarker, &status)
}

func (r *ReconcilePodProbeMarker) probe(podProbeMarker *appsv1alpha1.PodProbeMarker, pod *corev1.Pod) (probe.Result, error) {
	marker := podProbeMarker.Spec.MarkerProbe
	klog.V(4).Infof("debug: pod probe maker %v", podProbeMarker.Spec)

	if marker.Exec != nil {
		klog.V(3).Infof("Exec-Probe Pod: %v, Container: %v, Command: %v", pod.Name, marker.Exec.Container, marker.Exec.Command)
		execContainer := findContainerFromMarkProbe(pod, &marker)
		if execContainer == nil {
			return probe.Failure, fmt.Errorf("can't found container %s in pod %s", marker.Exec.Container, pod.Name)
		}
		return r.execProbe(pod, &marker)
	} else if marker.HTTPGet != nil {
		klog.V(3).Infof("find a http probe in pod %v", pod.Name)
		return r.httpProbe(pod, &marker)
	}

	return probe.Failure, fmt.Errorf("invalid mark probe method")
}

func (r *ReconcilePodProbeMarker) execProbe(pod *corev1.Pod, marker *appsv1alpha1.MarkProbe) (probe.Result, error) {
	klog.V(3).Infof("start exec probe in pod %v", pod.Name)
	kubeClient := genericclient.GetGenericClient().KubeClient
	timeout := time.Duration(marker.TimeoutSeconds) * time.Second

	req := kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Timeout(timeout).
		VersionedParams(&corev1.PodExecOptions{
			Container: marker.Exec.Container,
			Command:   marker.Exec.Command,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
			Stdin:     true,
		}, schemeutil.ParameterCodec)

	klog.V(3).Infof("build req object: %v, url: %v", req, req.URL().String())

	cfg, err := config.GetConfig()
	if err != nil {
		return probe.Failure, err
	}

	var stdout, stderr bytes.Buffer
	if err = execute("POST", req.URL(), cfg, nil, &stdout, &stderr, false); err != nil {
		return probe.Failure, err
	}

	return probe.Success, nil
}

func (r *ReconcilePodProbeMarker) httpProbe(pod *corev1.Pod, marker *appsv1alpha1.MarkProbe) (probe.Result, error) {
	timeout := time.Duration(marker.TimeoutSeconds) * time.Second
	url, err := buildURL(pod, marker.HTTPGet)
	if err != nil {
		return probe.Failure, err
	}
	headers := buildHeader(marker.HTTPGet)
	res, _, err := http.New().Probe(url, headers, timeout)
	return res, err
}

func (r *ReconcilePodProbeMarker) listPodFromProbeMarker(marker *appsv1alpha1.PodProbeMarker) ([]*corev1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(marker.Spec.Selector)
	if err != nil {
		return nil, err
	}
	podList := &corev1.PodList{}
	if err = r.List(context.TODO(), &client.ListOptions{
		LabelSelector: selector,
		Namespace:     marker.Namespace,
	}, podList); err != nil {
		return nil, err
	}

	var pods []*corev1.Pod
	for i, p := range podList.Items {
		if isRunningAndReady(&p) {
			pods = append(pods, &podList.Items[i])
		}
	}
	return pods, nil
}

func (r *ReconcilePodProbeMarker) syncStatus(podProbeMarker *appsv1alpha1.PodProbeMarker, status *appsv1alpha1.PodProbeMarkerStatus) error {
	newPodProbeMarker := podProbeMarker.DeepCopy()
	newPodProbeMarker.Status = *status
	return r.Update(context.TODO(), newPodProbeMarker)
}
