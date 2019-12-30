# PodProbeMarker

This gcontroller is used to label the pod. Like the [liveness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/). Through an http request or exec script, the pod is labeled according to the result.

## PodProbeMarker Spec

```
type PodProbeMarkerSpec struct {
	Selector    *metav1.LabelSelector `json:"selector"`
	Labels      map[string]string     `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	MarkerProbe MarkProbe             `json:"markProbe"`
}
```

`Selector`: it used to select which pods to probe.
`Labels`: it describe what labels are marked to pod.
`MarkerProbe` it used to exec probe.

```
type MarkProbe struct {
	Handler `json:",inline" protobuf:"bytes,1,opt,name=handler"`
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,3,opt,name=timeoutSeconds"`
	PeriodSeconds int32 `json:"periodSeconds,omitempty" protobuf:"varint,4,opt,name=periodSeconds"`
}

type Handler struct {
	Exec *ExecAction `json:"exec,omitempty" protobuf:"bytes,1,opt,name=exec"`
	HTTPGet *v1.HTTPGetAction `json:"httpGet,omitempty" protobuf:"bytes,2,opt,name=httpGet"`
}
```

## Example

### Http Probe

The `test-http.yaml` file below describes a podProbeMarker to label pod is-active by send a http request.

```
# test-http.yaml
apiVersion: apps.kruise.io/v1alpha1
kind: PodProbeMarker
metadata:
  name: test-http
spec:
  selector:
    matchLabels:
      app: nginx
  labels:
    app: nginx
    mark: is-active
  markProbe:
    httpGet:
      path: /
      port: 80
      httpHeaders:
      - name: Custom-Header
        value: Awesome
    initialDelaySeconds: 5
    periodSeconds: 5
```

### Exec Probe

The `test-exec.yaml` file below describes a podProbeMarker to label pod zk-slave by send exec a script.

```
# test-exec.yaml
apiVersion: apps.kruise.io/v1alpha1
kind: PodProbeMarker
metadata:
  name: test-exec
spec:
  selector:
    matchLabels:
      app: zookeeper
  labels:
    run: zk-slave
  markProbe:
    exec:
      command:
      - /root/is_zk_slave.sh
      container: zookeeper
    initialDelaySeconds: 5
    periodSeconds: 5
```
