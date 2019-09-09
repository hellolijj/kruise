# 统一灰度控制
> 整个通用灰度控制机制由2个部分组成，RolloutDefinition 和 RolloutControl。

## RolloutControl
  此控制器可以操作任何workload/operator的spec发布控制相关字段（例如pause、patition等）。按照 RolloutDefinition 所关联的资源类型的spec字段路径，通过修改相应资源的spec发布策略字段实现相应资源的控制。
### RolloutControl Spec
#### resource：
  `resource` 通过 apiVersion、kind、namespace、name 来定位要控制的资源实例

#### rolloutstrategy：
  `rolloutstrategy`表示设置的发布策略，包含 paused、partition、maxUnavailable，不过对于每一类资源实际所包含的控制策略取决于 RolloutDefinition 的设定。

### RolloutControl Status
  status 可以用来查看目前的实例的发布状态，包含 Replicas、CurrentReplicas、UpdatedReplicas、ReadyReplicas、ObservedGeneration 字段

### Examples
#### 发布控制
进行一个advanced statefulset的实例 nginx-sts 的发布，yaml描述如下：
（需要首先使用 RolloutDefinition 来进行字段关联）
```
apiVersion: apps.kruise.io/v1alpha1
kind: RolloutControl
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: rolloutcontrol-sts
spec:
  resource:
    kind: statefulsets
    apiVersion: apps.kruise.io/v1alpha1
    namespace: default
    name: nginx-sts
  rolloutstrategy:
      paused: false
      maxUnavailable: 30%
      partition: 3
```
进行控制发布，通过这个yaml文件：
```
# kubectl apply -f control-sts.yaml
```
#### 状态查询
通过查看 rolloutcontrol 的状态，看到实例更新情况
```
# kubectl get rolloutcontrols.apps.kruise.io/rolloutcontrol-sts
NAME                 CURRENT   UPDATED   READY   AGE
rolloutcontrol-sts   6         3         6       6m
```
## RolloutDefinition
  关联资源的发布控制字段。将 workload/operator 中发布策略字段的操作路径进行设置以供 RolloutControl 发布时利用。
### RolloutDefinition Spec
#### controlResource：
  `controlResource` 包含apiVersion和resource，用来定位 workload/operator 类型。
#### path：
`path` 包含specPath和statusPath。
specPath 用来描述关联controlResource的spec字段的操作路径。包含paused、partition、maxUnavailable。
statusPath 用来描述关联controlResource的status字段的操作路径。包含replicas、currentReplicas、readyReplicas、updatedReplicas、observedGeneration。
### Example
例子1：关联 Deployment 的发布控制策略：
```
# kubectl apply -f deployment_sample.yaml
```
```
apiVersion: apps.kruise.io/v1alpha1
kind: RolloutDefinition
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: rolloutdefinition-deployment
spec:
  controlResource:
    apiVersion:  apps/v1
    resource: deployments
  path:
    specPath:
      paused: spec.paused
      maxUnavailable: spec.strategy.rollingUpdate.maxUnavailable
    statusPath:
      replicas: status.replicas
      readyReplicas: status.readyReplicas
      currentReplicas: status.currentReplicas
      updatedReplicas: status.updatedReplicas
      observedGeneration: status.observedGeneration
      conditions: status.conditions
```
例子2：关联 Advanced StatusfulSet 的发布控制策略：
```
# kubectl apply -f statefulset_sample.yaml
```
```
apiVersion: apps.kruise.io/v1alpha1
kind: RolloutDefinition
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: rolloutdefinition-sts
spec:
  controlResource:
    apiVersion:  apps.kruise.io/v1alpha1
    resource: statefulsets
  path:
    specPath:
      paused: spec.updateStrategy.rollingUpdate.paused
      partition: spec.updateStrategy.rollingUpdate.partition
      maxUnavailable: spec.updateStrategy.rollingUpdate.maxUnavailable
    statusPath:
      replicas: status.replicas
      readyReplicas: status.readyReplicas
      currentReplicas: status.currentReplicas
      updatedReplicas: status.updatedReplicas
      observedGeneration: status.observedGeneration
      conditions: status.conditions
```