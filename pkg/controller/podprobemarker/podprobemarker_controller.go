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
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	genericclient "github.com/openkruise/kruise/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/probe"
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
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	prober, err := newProber()
	if err != nil {
		klog.V(5).Infof("new reconciler err: %v", err)
		return &ReconcilePodProbeMarker{}
	}

	return &ReconcilePodProbeMarker{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		prober: prober,
	}
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
	prober prober
}

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

	for _, p := range pods {
		klog.V(4).Info("find pod %v, to probe", p.Name)

		probeResult, _, err := r.runProbe(podProbeMarker, p)
		if err != nil || probeResult == probe.Failure {
			klog.Errorf("failed to run probe for podProbeMarker %s in pod %s, reason: %v", podProbeMarker.Name, p.Name, err)
			return reconcile.Result{}, err
		}

		klog.V(4).Infof("return probe result  %v", probeResult)
		if probeResult == probe.Success {
			newPod := updatePodLabels(p, podProbeMarker.Spec.Labels)
			_, err = r.prober.client.CoreV1().Pods(newPod.Namespace).Update(newPod)
			if err != nil {
				klog.Errorf("failed to update pod %v reason %v", newPod.Name, err)
				return reconcile.Result{}, err
			}
			klog.V(4).Infof("success in update pod %v", newPod.Name)

		}
	}

	klog.V(4).Infof("reconcile request: %v", request)

	probeTickerPeriod := time.Duration(podProbeMarker.Spec.MarkerProbe.PeriodSeconds) * time.Second
	if probeTickerPeriod != 0 {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: probeTickerPeriod,
		}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcilePodProbeMarker) listPodFromProbeMarker(marker *appsv1alpha1.PodProbeMarker) ([]*corev1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(marker.Spec.Selector)
	if err != nil {
		return nil, err
	}
	podList := &corev1.PodList{}
	err = r.List(context.TODO(), &client.ListOptions{
		Namespace:     marker.Namespace,
		LabelSelector: selector,
	}, podList)
	if err != nil {
		return nil, err
	}

	var pods []*corev1.Pod
	for _, p := range podList.Items {
		if isRunningAndReady(&p) {
			pods = append(pods, &p)
		}
	}
	return pods, nil
}

// func (pb *prober) runProbe(probeType probeType, p *v1.Probe, pod *v1.Pod, status v1.PodStatus, container v1.Container, containerID kubecontainer.ContainerID) (probe.Result, string, error) {
func (r *ReconcilePodProbeMarker) runProbe(podProbeMarker *appsv1alpha1.PodProbeMarker, pod *corev1.Pod) (probe.Result, string, error) {
	marker := podProbeMarker.Spec.MarkerProbe
	timeout := time.Duration(marker.TimeoutSeconds) * time.Second

	klog.V(4).Infof("debug: pod probe maker %v", podProbeMarker.Spec)

	if marker.Exec != nil {
		klog.V(4).Infof("Exec-Probe Pod: %v, Container: %v, Command: %v", pod.Name, marker.Exec.Container, marker.Exec.Command)
		execContainer := corev1.Container{}
		for _, c := range pod.Spec.Containers {
			if c.Name == marker.Exec.Container {
				execContainer = c
				break
			}
		}
		if len(execContainer.Name) == 0 {
			klog.Errorf("wrong pod name %v", podProbeMarker.Name)
		}
		return r.execProbe(podProbeMarker, pod, timeout)
	} else if marker.HTTPGet != nil {
		klog.V(4).Infof("find a http probe in pod %v", pod.Name)
		httpGet := podProbeMarker.Spec.MarkerProbe.HTTPGet
		scheme := strings.ToLower(string(httpGet.Scheme))
		if len(scheme) == 0 {
			scheme = "http"
		}
		host := httpGet.Host
		if host == "" {
			host = pod.Status.PodIP
		}
		// TODO: add condition for port
		port := int(httpGet.Port.IntVal)
		if port == 0 {
			klog.Errorf("failed to get proProbeMark port %s,", podProbeMarker.Name)
		}
		path := httpGet.Path
		klog.V(4).Infof("HTTP-Probe Host: %v://%v, Port: %v, Path: %v", scheme, host, port, path)
		url := formatURL(scheme, host, port, path)
		headers := buildHeader(httpGet.HTTPHeaders)
		klog.V(4).Infof("HTTP-Probe Headers: %v", headers)
		return r.prober.http.Probe(url, headers, timeout)
	} else {
		return probe.Failure, "invalid mark probe method only support exec and http", fmt.Errorf("invalid mark probe method")
	}

	return probe.Success, "", nil

}

func (r *ReconcilePodProbeMarker) execProbe(podProbeMarker *appsv1alpha1.PodProbeMarker, pod *corev1.Pod, timeout time.Duration) (probe.Result, string, error) {
	klog.V(4).Infof("start exec probe in pod %v", pod.Name)
	kubeClient := genericclient.GetGenericClient().KubeClient

	req := kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Timeout(timeout).
		VersionedParams(&corev1.PodExecOptions{
			Container: podProbeMarker.Spec.MarkerProbe.Exec.Container,
			Command:   podProbeMarker.Spec.MarkerProbe.Exec.Command,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
			Stdin:     true,
		}, scheme.ParameterCodec)

	klog.V(4).Infof("build req object: %v", req)
	klog.V(4).Infof("build req url: %v", req.URL().String())

	cfg, err := config.GetConfig()
	if err != nil {
		return probe.Failure, "unable to set up client config", err
	}

	var stdout, stderr bytes.Buffer
	// stdin := strings.NewReader(strings.Join(podProbeMarker.Spec.MarkerProbe.Exec.Command, " "))
	err = execute("POST", req.URL(), cfg, nil, &stdout, &stderr, false)
	if err != nil {
		klog.V(4).Infof("exec post error: %v", err)
	}

	return probe.Success, strings.TrimSpace(stdout.String()), err
}

// formatURL formats a URL from args.  For testability.
func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
	// Something is busted with the path, but it's too late to reject it. Pass it along as is.
	if err != nil {
		u = &url.URL{
			Path: path,
		}
	}
	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u
}

// buildHeaderMap takes a list of HTTPHeader <name, value> string
// pairs and returns a populated string->[]string http.Header map.
func buildHeader(headerList []corev1.HTTPHeader) http.Header {
	headers := make(http.Header)
	for _, header := range headerList {
		headers[header.Name] = append(headers[header.Name], header.Value)
	}
	return headers
}

//  update pod env with assigned status
func updatePodLabels(oldPod *corev1.Pod, labels map[string]string) (newPod *corev1.Pod) {
	newPod = oldPod.DeepCopy()
	if len(newPod.ObjectMeta.Labels) == 0 {
		newPod.ObjectMeta.Labels = map[string]string{}
	}
	for k, v := range labels {
		newPod.ObjectMeta.Labels[k] = v
	}
	klog.V(4).Infof("update new pod %v add labels: %v", oldPod.Name, labels)
	return newPod
}

func execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	klog.V(4).Infof("exec is  %v", exec)
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
