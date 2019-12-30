package podprobemarker

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubectl/util/podutils"
)

func isRunningAndReady(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodRunning && podutils.IsPodReady(pod)
}

func findContainerFromMarkProbe(pod *corev1.Pod, mark *appsv1alpha1.MarkProbe) *corev1.Container {
	var c *corev1.Container
	if pod == nil || mark == nil {
		return c
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == mark.Exec.Container {
			c = &container
			return c
		}
	}
	return c
}

func buildURL(pod *corev1.Pod, req *corev1.HTTPGetAction) (*url.URL, error) {
	scheme := strings.ToLower(string(req.Scheme))
	if len(scheme) == 0 {
		scheme = "http"
	}
	host := req.Host
	if host == "" {
		host = pod.Status.PodIP
	}
	port := req.Port.IntValue()
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("in valid port number %d", port)
	}
	path := req.Path
	klog.V(4).Infof("HTTP-Probe Host: %v://%v, Port: %v, Path: %v", scheme, host, port, path)
	return formatURL(scheme, host, port, path), nil
}

func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
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
func buildHeader(req *corev1.HTTPGetAction) http.Header {
	headerList := req.HTTPHeaders
	headers := make(http.Header)
	for _, header := range headerList {
		headers[header.Name] = append(headers[header.Name], header.Value)
	}
	return headers
}

func updatePodLabels(oldPod *corev1.Pod, labels map[string]string) (newPod *corev1.Pod) {
	newPod = oldPod.DeepCopy()
	if len(newPod.ObjectMeta.Labels) == 0 {
		newPod.ObjectMeta.Labels = map[string]string{}
	}
	for k, v := range labels {
		newPod.ObjectMeta.Labels[k] = v
	}
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
