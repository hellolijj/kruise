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
