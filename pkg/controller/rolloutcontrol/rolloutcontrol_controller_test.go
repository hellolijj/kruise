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

package rolloutcontrol

import (
	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"testing"
)

var (
	scheme *runtime.Scheme
)

func init() {
	scheme = runtime.NewScheme()
	_ = appsv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
}

// test rollout deploy pause, maxUnavailable
func TestRolloutControlDeployment(t *testing.T) {
	deployRolloutDef := newDeploymentRolloutDefinition("deploy-rollout-definition")
	deployment := newDeployment("nginx", 2)
	init := createReconcileRolloutControl(scheme, deployment, deployRolloutDef)
	latestDeploy, err := getLatestDeployment(init.Client, deployment)
	assert.NoError(t, err)
	assert.Equal(t, false, isDeploymentPuased(latestDeploy))

	deployRolloutControl := newDeploymentRolloutContorl("test-control-deploy-paused", "nginx", true, intstr.IntOrString{Type: intstr.String, StrVal: "45%"})
	reconcileRolloutControl := createReconcileRolloutControl(scheme, deployRolloutControl, deployRolloutDef, deployment)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: deployRolloutControl.Namespace,
			Name:      deployRolloutControl.Name,
		},
	}
	_, err = reconcileRolloutControl.Reconcile(request)
	assert.NoError(t, err)

	// check deploy pause
	latestDeploy, err = getLatestDeployment(reconcileRolloutControl.Client, deployment)
	assert.NoError(t, err)
	assert.Equal(t, true, isDeploymentPuased(latestDeploy))

	// checkout deploy maxUnavailable
	assert.Equal(t, "45%", getDeploymentMaxUnavailableStr(latestDeploy))
}

// test deployment status
func TestRolloutControlDeploymentStatus(t *testing.T) {
	deployRolloutDef := newDeploymentRolloutDefinition("deploy-rollout-definition")
	deployment := newDeployment("nginx", 2)
	deployment.Status = appsv1.DeploymentStatus{
		ObservedGeneration: 1,
		Replicas:           2,
		UpdatedReplicas:    3,
		ReadyReplicas:      4,
		Conditions:         nil,
		CollisionCount:     nil,
	}
	deployRolloutControl := newDeploymentRolloutContorl("test-control-deploy-paused", "nginx", true, intstr.IntOrString{Type: intstr.String, StrVal: "45%"})
	reconcileRolloutControl := createReconcileRolloutControl(scheme, deployRolloutControl, deployRolloutDef, deployment)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: deployRolloutControl.Namespace,
			Name:      deployRolloutControl.Name,
		},
	}
	_, err := reconcileRolloutControl.Reconcile(request)
	assert.NoError(t, err)

	// check deploy status
	latestRolloutControl, err := getLatestRolloutControl(reconcileRolloutControl.Client, deployRolloutControl)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), latestRolloutControl.Status.ObservedGeneration)
	assert.Equal(t, int32(2), latestRolloutControl.Status.Replicas)
	assert.Equal(t, int32(3), latestRolloutControl.Status.UpdatedReplicas)
	assert.Equal(t, int32(4), latestRolloutControl.Status.ReadyReplicas)
}

// test rollout statefulset partition、maxUnavailable
func TestRolloutControlStatefulset(t *testing.T) {
	statefulsetRolloutDef := newAdvaceStatefulsetRolloutDefinition("statefulset-rollout-definition")
	sts := newAdvancedStatefulset("sts-test", "nginx:1.7.9", 5)
	init := createReconcileRolloutControl(scheme, statefulsetRolloutDef, sts)
	updateSts := updateAdvancedStatefulsetSetPartition(sts, "nginx:1.8", 5)
	err := init.Update(context.TODO(), updateSts)
	assert.NoError(t, err)
	lastetSts, err := getLatestStatefulset(init.Client, sts)
	assert.NoError(t, err)
	assert.Equal(t, int32(5), *lastetSts.Spec.UpdateStrategy.RollingUpdate.Partition)

	stsRolloutControl := newStatefulsetRolloutContorl("test-rolloutControl", "sts-test", 2, intstr.IntOrString{Type: intstr.String, IntVal: 0, StrVal: "65%"})
	reconcileRolloutControl := createReconcileRolloutControl(scheme, stsRolloutControl, statefulsetRolloutDef, updateSts)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: stsRolloutControl.Namespace,
			Name:      stsRolloutControl.Name,
		},
	}
	_, err = reconcileRolloutControl.Reconcile(request)
	assert.NoError(t, err)

	// check image、 partition、 maxUnavailable
	lastetSts, err = getLatestStatefulset(reconcileRolloutControl.Client, sts)
	assert.NoError(t, err)
	assert.Equal(t, int32(2), getStatefulsetPartition(lastetSts))
	assert.Equal(t, "nginx:1.8", getStatefulsetImage(lastetSts))
	assert.Equal(t, "65%", getStatefulsetMaxUnavailableStr(lastetSts))
}

func TestRolloutControlStatefulsetStatus(t *testing.T) {
	statefulsetRolloutDef := newAdvaceStatefulsetRolloutDefinition("statefulset-rollout-definition")
	sts := newAdvancedStatefulset("sts-test", "nginx:1.7.9", 5)
	sts.Status = appsv1alpha1.StatefulSetStatus{
		ObservedGeneration: 5,
		Replicas:           6,
		ReadyReplicas:      7,
		UpdatedReplicas:    8,
		CollisionCount:     nil,
		Conditions:         nil,
	}

	stsRolloutControl := newStatefulsetRolloutContorl("test-rolloutControl", "sts-test", 2, intstr.IntOrString{Type: intstr.String, IntVal: 0, StrVal: "65%"})
	reconcileRolloutControl := createReconcileRolloutControl(scheme, stsRolloutControl, statefulsetRolloutDef, sts)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: stsRolloutControl.Namespace,
			Name:      stsRolloutControl.Name,
		},
	}

	_, err := reconcileRolloutControl.Reconcile(request)
	assert.NoError(t, err)

	// check image、 partition、 maxUnavailable
	latestRolloutControl, err := getLatestRolloutControl(reconcileRolloutControl.Client, stsRolloutControl)
	assert.NoError(t, err)

	assert.Equal(t, int64(5), latestRolloutControl.Status.ObservedGeneration)
	assert.Equal(t, int32(6), latestRolloutControl.Status.Replicas)
	assert.Equal(t, int32(7), latestRolloutControl.Status.ReadyReplicas)
	assert.Equal(t, int32(8), latestRolloutControl.Status.UpdatedReplicas)
}

// test updateControlWorkload
func TestSync(t *testing.T) {
	statefulsetRolloutDef := newAdvaceStatefulsetRolloutDefinition("statefulset-rollout-definition")
	sts := newAdvancedStatefulset("sts-test", "nginx:1.7.9", 5)

	init := createReconcileRolloutControl(scheme, statefulsetRolloutDef, sts)
	updateSts := updateAdvancedStatefulsetSetPartition(sts, "nginx:1.8", 5)
	err := init.Update(context.TODO(), updateSts)
	assert.NoError(t, err)
	lastetSts, err := getLatestStatefulset(init.Client, sts)
	assert.NoError(t, err)
	assert.Equal(t, int32(5), *lastetSts.Spec.UpdateStrategy.RollingUpdate.Partition)

	stsRolloutControl := newStatefulsetRolloutContorl("sts-rollout-control", "sts-test", 4, intstr.IntOrString{Type: intstr.String, IntVal: 0, StrVal: "65%"})
	reconcileRolloutControl := createReconcileRolloutControl(scheme, statefulsetRolloutDef, updateSts, stsRolloutControl)
	rolloutDef, err := reconcileRolloutControl.getDefFromControl(stsRolloutControl)
	assert.NoError(t, err)

	err = reconcileRolloutControl.sync(stsRolloutControl, rolloutDef)
	assert.NoError(t, err)

	lastetSts, err = getLatestStatefulset(reconcileRolloutControl.Client, sts)
	assert.NoError(t, err)
	assert.Equal(t, int32(4), getStatefulsetPartition(lastetSts))
	assert.Equal(t, "nginx:1.8", getStatefulsetImage(lastetSts))
}

// test getDefFromControl
func TestGetDefFromControl(t *testing.T) {
	rd := newDeploymentRolloutDefinition("deploy-definition")
	rcd := newDeploymentRolloutContorl("test-control-deploy", "nginx", true, intstr.IntOrString{Type: intstr.Int, IntVal: 2})

	// test exist
	reconcileRolloutControl := createReconcileRolloutControl(scheme, rd, rcd)
	findRd, err := reconcileRolloutControl.getDefFromControl(rcd)
	assert.NoError(t, err)
	assert.Equal(t, rd.Name, findRd.Name)
}

func createReconcileRolloutControl(scheme *runtime.Scheme, initObjs ...runtime.Object) ReconcileRolloutControl {
	fakeClient := fake.NewFakeClientWithScheme(scheme, initObjs...)
	reconcile := ReconcileRolloutControl{
		Client: fakeClient,
		scheme: scheme,
	}
	return reconcile
}

func newRolloutDefinition(name string, cr *appsv1alpha1.ControlResource, path *appsv1alpha1.Path) *appsv1alpha1.RolloutDefinition {
	if cr == nil || path == nil {
		return nil
	}

	rd := appsv1alpha1.RolloutDefinition{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps.kruise.io/v1alpha1", Kind: "RolloutDefinition"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: appsv1alpha1.RolloutDefinitionSpec{
			ControlResource: *cr,
			Path:            *path,
		},
	}
	return &rd
}

func newDeploymentRolloutDefinition(name string) *appsv1alpha1.RolloutDefinition {
	controlResouce := appsv1alpha1.ControlResource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
	}
	path := appsv1alpha1.Path{
		SpecPath: appsv1alpha1.SpecPath{
			Paused:         "spec.paused",
			MaxUnavailable: "spec.strategy.rollingUpdate.maxUnavailable",
		},
		StatusPath: appsv1alpha1.StatusPath{
			Replicas:           "status.replicas",
			ReadyReplicas:      "status.readyReplicas",
			UpdatedReplicas:    "status.updatedReplicas",
			ObservedGeneration: "status.observedGeneration",
			Conditions:         "status.conditions",
		},
	}
	return newRolloutDefinition(name, &controlResouce, &path)
}

func newAdvaceStatefulsetRolloutDefinition(name string) *appsv1alpha1.RolloutDefinition {
	controlResouce := appsv1alpha1.ControlResource{
		APIVersion: "apps.kruise.io/v1alpha1",
		Kind:       "Statefulset",
	}
	path := appsv1alpha1.Path{
		SpecPath: appsv1alpha1.SpecPath{
			Paused:         "spec.updateStrategy.rollingUpdate.paused",
			Partition:      "spec.updateStrategy.rollingUpdate.partition",
			MaxUnavailable: "spec.updateStrategy.rollingUpdate.maxUnavailable",
		},
		StatusPath: appsv1alpha1.StatusPath{
			Replicas:           "status.replicas",
			ReadyReplicas:      "status.readyReplicas",
			UpdatedReplicas:    "status.updatedReplicas",
			CurrentReplicas:    "status.currentReplicas",
			ObservedGeneration: "status.observedGeneration",
			Conditions:         "status.conditions",
		},
	}
	return newRolloutDefinition(name, &controlResouce, &path)
}

func newRolloutControl(name string, resource *appsv1alpha1.CompleteResource, strategy *appsv1alpha1.RolloutStrategy) *appsv1alpha1.RolloutControl {
	if resource == nil || strategy == nil {
		return nil
	}
	rc := appsv1alpha1.RolloutControl{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps.kruise.io/v1alpha1", Kind: "RolloutControl"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: appsv1alpha1.RolloutControlSpec{
			Resource:        *resource,
			RolloutStrategy: *strategy,
		},
	}
	return &rc
}

func newDeploymentRolloutContorl(name, controlDeployName string, pause bool, maxUnavailable intstr.IntOrString) *appsv1alpha1.RolloutControl {
	resouce := appsv1alpha1.CompleteResource{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		NameSpace:  metav1.NamespaceDefault,
		Name:       controlDeployName,
	}
	spec := appsv1alpha1.RolloutStrategy{
		Paused:         pause,
		MaxUnavailable: maxUnavailable,
	}
	return newRolloutControl(name, &resouce, &spec)
}

func newStatefulsetRolloutContorl(name, controlStatefulsetName string, partition int32, maxUnavailable intstr.IntOrString) *appsv1alpha1.RolloutControl {
	resouce := appsv1alpha1.CompleteResource{
		APIVersion: "apps.kruise.io/v1alpha1",
		Kind:       "Statefulset",
		NameSpace:  metav1.NamespaceDefault,
		Name:       controlStatefulsetName,
	}
	spec := appsv1alpha1.RolloutStrategy{
		Partition:      partition,
		MaxUnavailable: maxUnavailable,
	}
	return newRolloutControl(name, &resouce, &spec)
}

func newDeployment(name string, replicas int) *appsv1.Deployment {
	selector := map[string]string{"run": name}
	d := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func() *int32 { i := int32(replicas); return &i }(),
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "nginx:1.7.9",
						},
					},
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{},
					MaxSurge:       &intstr.IntOrString{},
				},
			},
		},
	}
	return &d
}

func getLatestDeployment(client client.Client, d *appsv1.Deployment) (*appsv1.Deployment, error) {
	newDeploy := &appsv1.Deployment{}
	deployKey := types.NamespacedName{
		Namespace: d.Namespace,
		Name:      d.Name,
	}
	err := client.Get(context.TODO(), deployKey, newDeploy)
	return newDeploy, err
}

func isDeploymentPuased(deploy *appsv1.Deployment) bool {
	return deploy.Spec.Paused == true
}

func getDeploymentMaxUnavailableStr(deploy *appsv1.Deployment) string {
	if deploy == nil || &deploy.Spec.Strategy == nil || deploy.Spec.Strategy.RollingUpdate == nil {
		return ""
	}

	return deploy.Spec.Strategy.RollingUpdate.MaxUnavailable.StrVal
}
func getStatefulsetMaxUnavailableStr(sts *appsv1alpha1.StatefulSet) string {
	if sts == nil || &sts.Spec.UpdateStrategy == nil || sts.Spec.UpdateStrategy.RollingUpdate == nil {
		return ""
	}

	return sts.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.StrVal
}

func newAdvancedStatefulset(name, image string, replicas int) *appsv1alpha1.StatefulSet {
	selector := map[string]string{"run": name}
	return &appsv1alpha1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps.kruise.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: appsv1alpha1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Replicas: func() *int32 { i := int32(replicas); return &i }(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: image,
						},
					},
				},
			},
			ServiceName: "governingsvc",
			UpdateStrategy: appsv1alpha1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1alpha1.RollingUpdateStatefulSetStrategy{
					Partition:      func() *int32 { i := int32(0); return &i }(),
					Paused:         false,
					MaxUnavailable: &intstr.IntOrString{},
				},
			},
			RevisionHistoryLimit: func() *int32 {
				limit := int32(2)
				return &limit
			}(),
		},
	}
}

func updateAdvancedStatefulsetSetPartition(sts *appsv1alpha1.StatefulSet, image string, partition int32) *appsv1alpha1.StatefulSet {
	updateSts := sts.DeepCopy()
	updateSts.Spec.UpdateStrategy.RollingUpdate.Partition = &partition
	updateSts.Spec.Template.Spec.Containers[0].Image = image
	return updateSts
}

func getLatestStatefulset(c client.Client, sts *appsv1alpha1.StatefulSet) (*appsv1alpha1.StatefulSet, error) {
	newSts := &appsv1alpha1.StatefulSet{}
	stsKey := types.NamespacedName{
		Namespace: sts.Namespace,
		Name:      sts.Name,
	}
	err := c.Get(context.TODO(), stsKey, newSts)
	return newSts, err
}

func getStatefulsetImage(sts *appsv1alpha1.StatefulSet) string {
	if sts == nil {
		return ""
	}
	return sts.Spec.Template.Spec.Containers[0].Image
}

func getStatefulsetPartition(sts *appsv1alpha1.StatefulSet) int32 {
	if sts == nil {
		return 0
	}
	return *sts.Spec.UpdateStrategy.RollingUpdate.Partition
}

func getLatestRolloutDef(c client.Client, rd *appsv1alpha1.RolloutDefinition) (*appsv1alpha1.RolloutDefinition, error) {
	newRd := &appsv1alpha1.RolloutDefinition{}
	stsKey := types.NamespacedName{
		Namespace: rd.Namespace,
		Name:      rd.Name,
	}
	err := c.Get(context.TODO(), stsKey, newRd)
	return newRd, err
}

func getLatestRolloutControl(c client.Client, rc *appsv1alpha1.RolloutControl) (*appsv1alpha1.RolloutControl, error) {
	newRc := &appsv1alpha1.RolloutControl{}
	rcKey := types.NamespacedName{
		Namespace: rc.Namespace,
		Name:      rc.Name,
	}
	err := c.Get(context.TODO(), rcKey, newRc)
	return newRc, err
}
