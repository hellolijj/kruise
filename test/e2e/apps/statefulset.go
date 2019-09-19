/*
Copyright 2019 The Kruise Authors.
Copyright 2014 The Kubernetes Authors.

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

package apps

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	kruiseclientset "github.com/openkruise/kruise/pkg/client/clientset/versioned"
	"github.com/openkruise/kruise/test/e2e/framework"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
)

const (
	zookeeperManifestPath   = "test/e2e/testing-manifests/statefulset/zookeeper"
	mysqlGaleraManifestPath = "test/e2e/testing-manifests/statefulset/mysql-galera"
	redisManifestPath       = "test/e2e/testing-manifests/statefulset/redis"
	cockroachDBManifestPath = "test/e2e/testing-manifests/statefulset/cockroachdb"
	// We don't restart MySQL cluster regardless of restartCluster, since MySQL doesn't handle restart well
	restartCluster = true

	// Timeout for reads from databases running on stateful pods.
	readTimeout = 60 * time.Second
)

// GCE Quota requirements: 3 pds, one per stateful pod manifest declared above.
// GCE Api requirements: nodes and master need storage r/w permissions.
var _ = SIGDescribe("StatefulSet", func() {
	f := framework.NewDefaultFramework("statefulset")
	var ns string
	var c clientset.Interface
	var kc kruiseclientset.Interface

	ginkgo.BeforeEach(func() {
		c = f.ClientSet
		kc = f.KruiseClientSet
		ns = f.Namespace.Name
	})

	framework.KruiseDescribe("Basic StatefulSet functionality [StatefulSetBasic]", func() {
		// ssName := "ss"
		labels := map[string]string{
			"foo": "bar",
			"baz": "blah",
		}
		headlessSvcName := "test"
		// var statefulPodMounts, podMounts []v1.VolumeMount
		// var ss *appsv1alpha1.StatefulSet

		ginkgo.BeforeEach(func() {
			// statefulPodMounts = []v1.VolumeMount{{Name: "datadir", MountPath: "/data/"}}
			// podMounts = []v1.VolumeMount{{Name: "home", MountPath: "/home"}}
			// ss = framework.NewStatefulSet(ssName, ns, headlessSvcName, 2, statefulPodMounts, podMounts, labels)

			ginkgo.By("Creating service " + headlessSvcName + " in namespace " + ns)
			headlessService := framework.CreateServiceSpec(headlessSvcName, "", true, labels)
			_, err := c.CoreV1().Services(ns).Create(headlessService)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			if ginkgo.CurrentGinkgoTestDescription().Failed {
				framework.DumpDebugInfo(c, ns)
			}
			framework.Logf("Deleting all statefulset in ns %v", ns)
			framework.DeleteAllStatefulSets(c, kc, ns)
		})

		// This can't be Conformance yet because it depends on a default
		// StorageClass and a dynamic provisioner.
		ginkgo.It("should provide basic identity", func() {
			// fmt.Println(ss)
			// ginkgo.By("Creating statefulset " + ssName + " in namespace " + ns)
			// *(ss.Spec.Replicas) = 3
			// sst := framework.NewStatefulSetTester(c, kc)
			// sst.PauseNewPods(ss)

			// _, err := kc.AppsV1alpha1().StatefulSets(ns).Create(ss)
			// gomega.Expect(err).NotTo(gomega.HaveOccurred())
			//
			// ginkgo.By("Saturating stateful set " + ss.Name)
			// sst.Saturate(ss)
			//
			// ginkgo.By("Verifying statefulset mounted data directory is usable")
			// framework.ExpectNoError(sst.CheckMount(ss, "/data"))
			//
			// ginkgo.By("Verifying statefulset provides a stable hostname for each pod")
			// framework.ExpectNoError(sst.CheckHostname(ss))
			//
			// ginkgo.By("Verifying statefulset set proper service name")
			// framework.ExpectNoError(sst.CheckServiceName(ss, headlessSvcName))
			//
			// cmd := "echo $(hostname) | dd of=/data/hostname conv=fsync"
			// ginkgo.By("Running " + cmd + " in all stateful pods")
			// framework.ExpectNoError(sst.ExecInStatefulPods(ss, cmd))
			//
			// ginkgo.By("Restarting statefulset " + ss.Name)
			// sst.Restart(ss)
			// sst.WaitForRunningAndReady(*ss.Spec.Replicas, ss)
			//
			// ginkgo.By("Verifying statefulset mounted data directory is usable")
			// framework.ExpectNoError(sst.CheckMount(ss, "/data"))
			//
			// cmd = "if [ \"$(cat /data/hostname)\" = \"$(hostname)\" ]; then exit 0; else exit 1; fi"
			// ginkgo.By("Running " + cmd + " in all stateful pods")
			// framework.ExpectNoError(sst.ExecInStatefulPods(ss, cmd))
		})
		
	})

	framework.KruiseDescribe("Deploy clustered applications [Feature:StatefulSet] [Slow]", func() {
		var sst *framework.StatefulSetTester
		var appTester *clusterAppTester

		ginkgo.BeforeEach(func() {
			sst = framework.NewStatefulSetTester(c, kc)
			appTester = &clusterAppTester{tester: sst, ns: ns}
		})

		ginkgo.AfterEach(func() {
			if ginkgo.CurrentGinkgoTestDescription().Failed {
				framework.DumpDebugInfo(c, ns)
			}
			framework.Logf("Deleting all statefulset in ns %v", ns)
			framework.DeleteAllStatefulSets(c, kc, ns)
		})

		// Do not mark this as Conformance.
		// StatefulSet Conformance should not be dependent on specific applications.
		ginkgo.It("should creating a working zookeeper cluster", func() {
			appTester.statefulPod = &zookeeperTester{tester: sst}
			appTester.run()
		})
	})
})

func kubectlExecWithRetries(args ...string) (out string) {
	var err error
	for i := 0; i < 3; i++ {
		if out, err = framework.RunKubectl(args...); err == nil {
			return
		}
		framework.Logf("Retrying %v:\nerror %v\nstdout %v", args, err, out)
	}
	framework.Failf("Failed to execute \"%v\" with retries: %v", args, err)
	return
}

type statefulPodTester interface {
	deploy(ns string) *appsv1alpha1.StatefulSet
	write(statefulPodIndex int, kv map[string]string)
	read(statefulPodIndex int, key string) string
	name() string
}

type clusterAppTester struct {
	ns          string
	statefulPod statefulPodTester
	tester      *framework.StatefulSetTester
}

func (c *clusterAppTester) run() {
	ginkgo.By("Deploying " + c.statefulPod.name())
	ss := c.statefulPod.deploy(c.ns)

	ginkgo.By("Creating foo:bar in member with index 0")
	c.statefulPod.write(0, map[string]string{"foo": "bar"})

	switch c.statefulPod.(type) {
	case *mysqlGaleraTester:
		// Don't restart MySQL cluster since it doesn't handle restarts well
	default:
		if restartCluster {
			ginkgo.By("Restarting stateful set " + ss.Name)
			c.tester.Restart(ss)
			c.tester.WaitForRunningAndReady(*ss.Spec.Replicas, ss)
		}
	}

	ginkgo.By("Reading value under foo from member with index 2")
	if err := pollReadWithTimeout(c.statefulPod, 2, "foo", "bar"); err != nil {
		framework.Failf("%v", err)
	}
}

type zookeeperTester struct {
	ss     *appsv1alpha1.StatefulSet
	tester *framework.StatefulSetTester
}

func (z *zookeeperTester) name() string {
	return "zookeeper"
}

func (z *zookeeperTester) deploy(ns string) *appsv1alpha1.StatefulSet {
	z.ss = z.tester.CreateStatefulSet(zookeeperManifestPath, ns)
	return z.ss
}

func (z *zookeeperTester) write(statefulPodIndex int, kv map[string]string) {
	name := fmt.Sprintf("%v-%d", z.ss.Name, statefulPodIndex)
	ns := fmt.Sprintf("--namespace=%v", z.ss.Namespace)
	for k, v := range kv {
		cmd := fmt.Sprintf("/opt/zookeeper/bin/zkCli.sh create /%v %v", k, v)
		framework.Logf(framework.RunKubectlOrDie("exec", ns, name, "--", "/bin/sh", "-c", cmd))
	}
}

func (z *zookeeperTester) read(statefulPodIndex int, key string) string {
	name := fmt.Sprintf("%v-%d", z.ss.Name, statefulPodIndex)
	ns := fmt.Sprintf("--namespace=%v", z.ss.Namespace)
	cmd := fmt.Sprintf("/opt/zookeeper/bin/zkCli.sh get /%v", key)
	return lastLine(framework.RunKubectlOrDie("exec", ns, name, "--", "/bin/sh", "-c", cmd))
}

type mysqlGaleraTester struct {
	ss     *appsv1alpha1.StatefulSet
	tester *framework.StatefulSetTester
}

func (m *mysqlGaleraTester) name() string {
	return "mysql: galera"
}

func (m *mysqlGaleraTester) mysqlExec(cmd, ns, podName string) string {
	cmd = fmt.Sprintf("/usr/bin/mysql -u root -B -e '%v'", cmd)
	// TODO: Find a readiness probe for mysql that guarantees writes will
	// succeed and ditch retries. Current probe only reads, so there's a window
	// for a race.
	return kubectlExecWithRetries(fmt.Sprintf("--namespace=%v", ns), "exec", podName, "--", "/bin/sh", "-c", cmd)
}

func (m *mysqlGaleraTester) deploy(ns string) *appsv1alpha1.StatefulSet {
	m.ss = m.tester.CreateStatefulSet(mysqlGaleraManifestPath, ns)

	framework.Logf("Deployed statefulset %v, initializing database", m.ss.Name)
	for _, cmd := range []string{
		"create database statefulset;",
		"use statefulset; create table foo (k varchar(20), v varchar(20));",
	} {
		framework.Logf(m.mysqlExec(cmd, ns, fmt.Sprintf("%v-0", m.ss.Name)))
	}
	return m.ss
}

func (m *mysqlGaleraTester) write(statefulPodIndex int, kv map[string]string) {
	name := fmt.Sprintf("%v-%d", m.ss.Name, statefulPodIndex)
	for k, v := range kv {
		cmd := fmt.Sprintf("use statefulset; insert into foo (k, v) values (\"%v\", \"%v\");", k, v)
		framework.Logf(m.mysqlExec(cmd, m.ss.Namespace, name))
	}
}

func (m *mysqlGaleraTester) read(statefulPodIndex int, key string) string {
	name := fmt.Sprintf("%v-%d", m.ss.Name, statefulPodIndex)
	return lastLine(m.mysqlExec(fmt.Sprintf("use statefulset; select v from foo where k=\"%v\";", key), m.ss.Namespace, name))
}

type redisTester struct {
	ss     *appsv1alpha1.StatefulSet
	tester *framework.StatefulSetTester
}

func (m *redisTester) name() string {
	return "redis: master/slave"
}

func (m *redisTester) redisExec(cmd, ns, podName string) string {
	cmd = fmt.Sprintf("/opt/redis/redis-cli -h %v %v", podName, cmd)
	return framework.RunKubectlOrDie(fmt.Sprintf("--namespace=%v", ns), "exec", podName, "--", "/bin/sh", "-c", cmd)
}

func (m *redisTester) deploy(ns string) *appsv1alpha1.StatefulSet {
	m.ss = m.tester.CreateStatefulSet(redisManifestPath, ns)
	return m.ss
}

func (m *redisTester) write(statefulPodIndex int, kv map[string]string) {
	name := fmt.Sprintf("%v-%d", m.ss.Name, statefulPodIndex)
	for k, v := range kv {
		framework.Logf(m.redisExec(fmt.Sprintf("SET %v %v", k, v), m.ss.Namespace, name))
	}
}

func (m *redisTester) read(statefulPodIndex int, key string) string {
	name := fmt.Sprintf("%v-%d", m.ss.Name, statefulPodIndex)
	return lastLine(m.redisExec(fmt.Sprintf("GET %v", key), m.ss.Namespace, name))
}

type cockroachDBTester struct {
	ss     *appsv1alpha1.StatefulSet
	tester *framework.StatefulSetTester
}

func (c *cockroachDBTester) name() string {
	return "CockroachDB"
}

func (c *cockroachDBTester) cockroachDBExec(cmd, ns, podName string) string {
	cmd = fmt.Sprintf("/cockroach/cockroach sql --insecure --host %s.cockroachdb -e \"%v\"", podName, cmd)
	return framework.RunKubectlOrDie(fmt.Sprintf("--namespace=%v", ns), "exec", podName, "--", "/bin/sh", "-c", cmd)
}

func (c *cockroachDBTester) deploy(ns string) *appsv1alpha1.StatefulSet {
	c.ss = c.tester.CreateStatefulSet(cockroachDBManifestPath, ns)
	framework.Logf("Deployed statefulset %v, initializing database", c.ss.Name)
	for _, cmd := range []string{
		"CREATE DATABASE IF NOT EXISTS foo;",
		"CREATE TABLE IF NOT EXISTS foo.bar (k STRING PRIMARY KEY, v STRING);",
	} {
		framework.Logf(c.cockroachDBExec(cmd, ns, fmt.Sprintf("%v-0", c.ss.Name)))
	}
	return c.ss
}

func (c *cockroachDBTester) write(statefulPodIndex int, kv map[string]string) {
	name := fmt.Sprintf("%v-%d", c.ss.Name, statefulPodIndex)
	for k, v := range kv {
		cmd := fmt.Sprintf("UPSERT INTO foo.bar VALUES ('%v', '%v');", k, v)
		framework.Logf(c.cockroachDBExec(cmd, c.ss.Namespace, name))
	}
}
func (c *cockroachDBTester) read(statefulPodIndex int, key string) string {
	name := fmt.Sprintf("%v-%d", c.ss.Name, statefulPodIndex)
	return lastLine(c.cockroachDBExec(fmt.Sprintf("SELECT v FROM foo.bar WHERE k='%v';", key), c.ss.Namespace, name))
}

func lastLine(out string) string {
	outLines := strings.Split(strings.Trim(out, "\n"), "\n")
	return outLines[len(outLines)-1]
}

func pollReadWithTimeout(statefulPod statefulPodTester, statefulPodNumber int, key, expectedVal string) error {
	err := wait.PollImmediate(time.Second, readTimeout, func() (bool, error) {
		val := statefulPod.read(statefulPodNumber, key)
		if val == "" {
			return false, nil
		} else if val != expectedVal {
			return false, fmt.Errorf("expected value %v, found %v", expectedVal, val)
		}
		return true, nil
	})

	if err == wait.ErrWaitTimeout {
		return fmt.Errorf("timed out when trying to read value for key %v from stateful pod %d", key, statefulPodNumber)
	}
	return err
}

// This function is used by two tests to test StatefulSet rollbacks: one using
// PVCs and one using no storage.
func rollbackTest(c clientset.Interface, kc kruiseclientset.Interface, ns string, ss *appsv1alpha1.StatefulSet) {
	sst := framework.NewStatefulSetTester(c, kc)
	sst.SetHTTPProbe(ss)
	ss, err := kc.AppsV1alpha1().StatefulSets(ns).Create(ss)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	sst.WaitForRunningAndReady(*ss.Spec.Replicas, ss)
	ss = sst.WaitForStatus(ss)
	currentRevision, updateRevision := ss.Status.CurrentRevision, ss.Status.UpdateRevision
	gomega.Expect(currentRevision).To(gomega.Equal(updateRevision),
		fmt.Sprintf("StatefulSet %s/%s created with update revision %s not equal to current revision %s",
			ss.Namespace, ss.Name, updateRevision, currentRevision))
	pods := sst.GetPodList(ss)
	for i := range pods.Items {
		gomega.Expect(pods.Items[i].Labels[apps.StatefulSetRevisionLabel]).To(gomega.Equal(currentRevision),
			fmt.Sprintf("Pod %s/%s revision %s is not equal to current revision %s",
				pods.Items[i].Namespace,
				pods.Items[i].Name,
				pods.Items[i].Labels[apps.StatefulSetRevisionLabel],
				currentRevision))
	}
	sst.SortStatefulPods(pods)
	err = sst.BreakPodHTTPProbe(ss, &pods.Items[1])
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ss, pods = sst.WaitForPodNotReady(ss, pods.Items[1].Name)
	newImage := NewNginxImage
	oldImage := ss.Spec.Template.Spec.Containers[0].Image

	ginkgo.By(fmt.Sprintf("Updating StatefulSet template: update image from %s to %s", oldImage, newImage))
	gomega.Expect(oldImage).NotTo(gomega.Equal(newImage), "Incorrect test setup: should update to a different image")
	ss, err = framework.UpdateStatefulSetWithRetries(kc, ns, ss.Name, func(update *appsv1alpha1.StatefulSet) {
		update.Spec.Template.Spec.Containers[0].Image = newImage
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	ginkgo.By("Creating a new revision")
	ss = sst.WaitForStatus(ss)
	currentRevision, updateRevision = ss.Status.CurrentRevision, ss.Status.UpdateRevision
	gomega.Expect(currentRevision).NotTo(gomega.Equal(updateRevision),
		"Current revision should not equal update revision during rolling update")

	ginkgo.By("Updating Pods in reverse ordinal order")
	pods = sst.GetPodList(ss)
	sst.SortStatefulPods(pods)
	err = sst.RestorePodHTTPProbe(ss, &pods.Items[1])
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ss, pods = sst.WaitForPodReady(ss, pods.Items[1].Name)
	ss, pods = sst.WaitForRollingUpdate(ss)
	gomega.Expect(ss.Status.CurrentRevision).To(gomega.Equal(updateRevision),
		fmt.Sprintf("StatefulSet %s/%s current revision %s does not equal update revision %s on update completion",
			ss.Namespace,
			ss.Name,
			ss.Status.CurrentRevision,
			updateRevision))
	for i := range pods.Items {
		gomega.Expect(pods.Items[i].Spec.Containers[0].Image).To(gomega.Equal(newImage),
			fmt.Sprintf(" Pod %s/%s has image %s not have new image %s",
				pods.Items[i].Namespace,
				pods.Items[i].Name,
				pods.Items[i].Spec.Containers[0].Image,
				newImage))
		gomega.Expect(pods.Items[i].Labels[apps.StatefulSetRevisionLabel]).To(gomega.Equal(updateRevision),
			fmt.Sprintf("Pod %s/%s revision %s is not equal to update revision %s",
				pods.Items[i].Namespace,
				pods.Items[i].Name,
				pods.Items[i].Labels[apps.StatefulSetRevisionLabel],
				updateRevision))
	}

	ginkgo.By("Rolling back to a previous revision")
	err = sst.BreakPodHTTPProbe(ss, &pods.Items[1])
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ss, pods = sst.WaitForPodNotReady(ss, pods.Items[1].Name)
	priorRevision := currentRevision
	ss, err = framework.UpdateStatefulSetWithRetries(kc, ns, ss.Name, func(update *appsv1alpha1.StatefulSet) {
		update.Spec.Template.Spec.Containers[0].Image = oldImage
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ss = sst.WaitForStatus(ss)
	currentRevision, updateRevision = ss.Status.CurrentRevision, ss.Status.UpdateRevision
	gomega.Expect(currentRevision).NotTo(gomega.Equal(updateRevision),
		"Current revision should not equal update revision during roll back")
	gomega.Expect(priorRevision).To(gomega.Equal(updateRevision),
		"Prior revision should equal update revision during roll back")

	ginkgo.By("Rolling back update in reverse ordinal order")
	pods = sst.GetPodList(ss)
	sst.SortStatefulPods(pods)
	sst.RestorePodHTTPProbe(ss, &pods.Items[1])
	ss, pods = sst.WaitForPodReady(ss, pods.Items[1].Name)
	ss, pods = sst.WaitForRollingUpdate(ss)
	gomega.Expect(ss.Status.CurrentRevision).To(gomega.Equal(priorRevision),
		fmt.Sprintf("StatefulSet %s/%s current revision %s does not equal prior revision %s on rollback completion",
			ss.Namespace,
			ss.Name,
			ss.Status.CurrentRevision,
			updateRevision))

	for i := range pods.Items {
		gomega.Expect(pods.Items[i].Spec.Containers[0].Image).To(gomega.Equal(oldImage),
			fmt.Sprintf("Pod %s/%s has image %s not equal to previous image %s",
				pods.Items[i].Namespace,
				pods.Items[i].Name,
				pods.Items[i].Spec.Containers[0].Image,
				oldImage))
		gomega.Expect(pods.Items[i].Labels[apps.StatefulSetRevisionLabel]).To(gomega.Equal(priorRevision),
			fmt.Sprintf("Pod %s/%s revision %s is not equal to prior revision %s",
				pods.Items[i].Namespace,
				pods.Items[i].Name,
				pods.Items[i].Labels[apps.StatefulSetRevisionLabel],
				priorRevision))
	}
}
