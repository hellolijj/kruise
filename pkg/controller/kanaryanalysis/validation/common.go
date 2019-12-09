package validation

import (
	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"time"
)

// IsValidationStartLine returns true if the InitialDelay validation is over.
func IsValidationStartLine(ka *v1alpha1.KanaryAnalysis) bool {

	if ka.Spec.Validation.InitialDelay == nil {
		return true
	}

	return ka.CreationTimestamp.Time.Add(ka.Spec.Validation.InitialDelay.Duration).Before(time.Now())
}

// true 表示已经超过了 deadline
func IsDeadlinePeriodDone(ka *v1alpha1.KanaryAnalysis) bool {
	if ka.Spec.Validation.ValidationPeriod == nil {
		return false
	}

	create := ka.CreationTimestamp.Time
	if ka.Spec.Validation.InitialDelay != nil {
		create = create.Add(ka.Spec.Validation.InitialDelay.Duration)
	}

	return create.Add(ka.Spec.Validation.ValidationPeriod.Duration).Before(time.Now())
}
