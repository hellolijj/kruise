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

package validating

import (
	"context"
	"github.com/openkruise/kruise/pkg/util/gate"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"net/http"
	"regexp"

	genericvalidation "k8s.io/apimachinery/pkg/api/validation"
	validationutil "k8s.io/apimachinery/pkg/util/validation"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

func init() {
	webhookName := "validating-create-rolloutdefinition"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &RolloutDefinitionCreateHandler{})
}

const (
	rolloutDefinitionNameMaxLen = 63
)

var (
	validateRolloutDefinitionNameMsg = "RolloutDefinition name must consist of alphanumeric characters or '-'"
	validateRolloutDefinitionRegex   = regexp.MustCompile(validRolloutDefinitionNameFmt)
	validRolloutDefinitionNameFmt    = `^[a-zA-Z0-9\-]+$`
)

// RolloutDefinitionCreateHandler handles RolloutDefinition
type RolloutDefinitionCreateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// Client  client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func validatingRolloutDefinitionFn(ctx context.Context, obj *appsv1alpha1.RolloutDefinition) (bool, string, error) {
	allErrs := validateRolloutDefinition(obj)
	if len(allErrs) != 0 {
		return false, "", allErrs.ToAggregate()
	}
	return true, "allowed to be admitted", nil
}

func validateRolloutDefinition(obj *appsv1alpha1.RolloutDefinition) field.ErrorList {
	klog.V(4).Info(obj.ObjectMeta)
	allErrs := genericvalidation.ValidateObjectMeta(&obj.ObjectMeta, true, validateRolloutDefinitionName, field.NewPath("metadata"))
	allErrs = append(allErrs, validateRolloutDefinitionSpec(&obj.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateRolloutDefinitionName(name string, prefix bool) (allErrs []string) {
	if !validateRolloutDefinitionRegex.MatchString(name) {
		allErrs = append(allErrs, validationutil.RegexError(validateRolloutDefinitionNameMsg, validRolloutDefinitionNameFmt, "example-com"))
	}
	if len(name) > rolloutDefinitionNameMaxLen {
		allErrs = append(allErrs, validationutil.MaxLenError(rolloutDefinitionNameMaxLen))
	}
	return allErrs
}

func validateRolloutDefinitionSpec(spec *appsv1alpha1.RolloutDefinitionSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// checkout resouce api and kind to get

	// validate whether resource field complete
	if spec.ControlResource.APIVersion == "" || spec.ControlResource.Kind == "" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resource"), spec.ControlResource, "control resource should be complete"))
	}
	obj := unstructured.Unstructured{}
	obj.SetAPIVersion(spec.ControlResource.APIVersion)
	obj.SetKind(spec.ControlResource.Kind)

	// to get gvk and checkout we
	gvk, err := apiutil.GVKForObject(&obj, runtime.NewScheme())
	klog.V(4).Info(gvk)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resource"), spec.ControlResource, err.Error()))
	}
	if !gate.DiscoveryEnabled(gvk) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resource"), spec.ControlResource, "can't find resource by .spec.ControlResource fields"))
	}

	return allErrs
}

var _ admission.Handler = &RolloutDefinitionCreateHandler{}

// Handle handles admission requests.
func (h *RolloutDefinitionCreateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &appsv1alpha1.RolloutDefinition{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := validatingRolloutDefinitionFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

//var _ inject.Client = &RolloutDefinitionCreateHandler{}
//
//// InjectClient injects the client into the RolloutDefinitionCreateHandler
//func (h *RolloutDefinitionCreateHandler) InjectClient(c client.Client) error {
//	h.Client = c
//	return nil
//}

var _ inject.Decoder = &RolloutDefinitionCreateHandler{}

// InjectDecoder injects the decoder into the RolloutDefinitionCreateHandler
func (h *RolloutDefinitionCreateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
