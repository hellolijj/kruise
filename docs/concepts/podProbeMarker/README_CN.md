# PodProbeMarker

这个控制器是用来对 pod 打标签的。类似于 ![liveness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) 通过一条 http get 请求，或者执行一个 exec 脚本，然后根据执行的结果对 pod 搭上预定对标签。

## PodProbeMarker Spec

```
type PodProbeMarkerSpec struct {
	Selector    *metav1.LabelSelector `json:"selector"`
	Labels      map[string]string     `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	MarkerProbe MarkProbe             `json:"markProbe"`
}
```

`Selector` 用于选择哪些标签进行打标
`Labels` 用于对结果搭上什么样对标签
`MarkerProbe` 用于执行probe操作


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

如下 `test-http.yaml` 文件通过 http 请求来给 pod 打上标签.

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

如下 `test-exec.yaml` 文件通过 脚本 请求来给 pod 打上标签.

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