package types

import v1 "k8s.io/api/core/v1"

type ValidateResult struct {
	IsFailed bool
	Reason   string
}

// hasTempWorkload deployment 在更新过程中会创建 rs 作为 old rs, 完成发布之后 删除 rs.
type ToCheckPods struct {
	Pods []*v1.Pod
	Replicas int
	HasTempWorkload bool
}

