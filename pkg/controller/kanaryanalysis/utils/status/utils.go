package status

import (
	"context"
	"fmt"
	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// UpdateKanaryDeploymentStatus used to update the KanaryDeployment.Status if it has changed.
func UpdateKanaryAnalysisStatus(client client.Client, ka *v1alpha1.KanaryAnalysis, conditionType v1alpha1.KanaryAnalysisConditionType, reason string) error {
	newStatus := ka.Status.DeepCopy()

	// update condition
	updateKanaryAnalysisStatusConditions(newStatus, conditionType, v1.ConditionTrue, reason, true)
	if &ka.Status == newStatus {
		return nil
	}

	// update report
	updateStatusReport(ka, newStatus)
	if !equality.Semantic.DeepEqual(&ka.Status, newStatus) {
		updatedKd := ka.DeepCopy()
		updatedKd.Status = *newStatus
		err2 := client.Update(context.TODO(), updatedKd)
		if err2 != nil {
			return fmt.Errorf("failed to update KanaryAnalysisStatus status KanaryDeployment.Namespace %s KanaryDeployment.Name %s for reason %s", updatedKd.Name, updatedKd.Namespace, err2)
		}
	}
	return nil
}

// status 是deep来的， t 是 fail, condition status 是 true,
func updateKanaryAnalysisStatusConditions(status *v1alpha1.KanaryAnalysisStatus, t v1alpha1.KanaryAnalysisConditionType, conditionStatus v1.ConditionStatus, desc string, writeFalseIfNotExist bool) {
	idConditionComplete := getIndexForConditionType(status, t)
	if idConditionComplete >= 0 {
		if status.Conditions[idConditionComplete].Status != conditionStatus {
			status.Conditions[idConditionComplete].LastTransitionTime = metav1.Now()
			status.Conditions[idConditionComplete].Status = conditionStatus
		}
		status.Conditions[idConditionComplete].LastUpdateTime = metav1.Now()
		status.Conditions[idConditionComplete].Message = desc
	} else {
		// Only add if the condition is True
		status.Conditions = append(status.Conditions, NewKanaryAnalysisStatusCondition(t, conditionStatus, desc, desc))
	}
}

// NewKanaryDeploymentStatusCondition returns new KanaryDeploymentCondition instance
func NewKanaryAnalysisStatusCondition(conditionType v1alpha1.KanaryAnalysisConditionType, conditionStatus v1.ConditionStatus, reason, message string) v1alpha1.KanaryAnalysisCondition {
	return v1alpha1.KanaryAnalysisCondition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func getIndexForConditionType(status *v1alpha1.KanaryAnalysisStatus, t v1alpha1.KanaryAnalysisConditionType) int {
	idCondition := -1
	if status == nil {
		return idCondition
	}
	for i, condition := range status.Conditions {
		if condition.Type == t {
			idCondition = i
			break
		}
	}
	return idCondition
}

func updateStatusReport(ka *v1alpha1.KanaryAnalysis, status *v1alpha1.KanaryAnalysisStatus) {
	status.Report = v1alpha1.KanaryAnalysisStatusReport{
		Status:     getReportStatus(status),
		Validation: getValidation(ka),
		Workload:   getWorkload(ka),
	}
}

func getValidation(ka *v1alpha1.KanaryAnalysis) string {
	var list []string
	for _, v := range ka.Spec.Validation.Items {
		if v.LabelWatch != nil {
			list = append(list, "labelWatch")
		}
		if v.PromQL != nil {
			list = append(list, "promQL")
		}
		if v.Manual != nil {
			list = append(list, "manual")
		}
	}
	if len(list) == 0 {
		return "unknow"
	}
	return strings.Join(list, ",")
}

func getWorkload(ka *v1alpha1.KanaryAnalysis) string {
	workload := ka.Spec.Workload
	return workload.Kind + "." + workload.APIVersion + "/" + workload.Name
}

func getReportStatus(status *v1alpha1.KanaryAnalysisStatus) string {

	if IsKanaryAnalysisFailed(status) {
		return string(v1alpha1.FailedKanaryAnalysisConditionType)
	}

	if IsWorkloadKanaryAnalysisUpdated(status) {
		return string(v1alpha1.WorkloadUpdatedKanaryAnalysisConditionType)
	}

	if IsKanaryAnalysisRunning(status) {
		return string(v1alpha1.RunningKanaryAnalysisConditionType)
	}

	if isKanaryAnalysisScheduled(status) {
		return string(v1alpha1.ScheduledKanaryAnalysisConditionType)
	}

	if isKanaryAnalysisErrored(status) {
		return string(v1alpha1.ErroredKanaryAnalysisConditionType)
	}

	return "-"
}

func IsKanaryAnalysisFailed(status *v1alpha1.KanaryAnalysisStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, v1alpha1.FailedKanaryAnalysisConditionType)
	if id >= 0 && status.Conditions[id].Status == v1.ConditionTrue {
		return true
	}
	return false
}

func GetKanaryAnalysisFailedReason(status *v1alpha1.KanaryAnalysisStatus) string {
	if status == nil {
		return ""
	}
	id := getIndexForConditionType(status, v1alpha1.FailedKanaryAnalysisConditionType)
	if id >= 0 && status.Conditions[id].Status == v1.ConditionTrue {
		return status.Conditions[id].Reason
	}
	return ""
}

func IsWorkloadKanaryAnalysisUpdated(status *v1alpha1.KanaryAnalysisStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, v1alpha1.WorkloadUpdatedKanaryAnalysisConditionType)
	if id >= 0 && status.Conditions[id].Status == v1.ConditionTrue {
		return true
	}
	return false
}

func IsKanaryAnalysisRunning(status *v1alpha1.KanaryAnalysisStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, v1alpha1.RunningKanaryAnalysisConditionType)
	if id >= 0 && status.Conditions[id].Status == v1.ConditionTrue {
		return true
	}
	return false
}

func isKanaryAnalysisScheduled(status *v1alpha1.KanaryAnalysisStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, v1alpha1.ScheduledKanaryAnalysisConditionType)
	if id >= 0 && status.Conditions[id].Status == v1.ConditionTrue {
		return true
	}
	return false
}

func isKanaryAnalysisErrored(status *v1alpha1.KanaryAnalysisStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, v1alpha1.ErroredKanaryAnalysisConditionType)
	if id >= 0 && status.Conditions[id].Status == v1.ConditionTrue {
		return true
	}
	return false
}
