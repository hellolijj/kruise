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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"net/http"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	namespacestype "k8s.io/apimachinery/pkg/types"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	genericvalidation "k8s.io/apimachinery/pkg/api/validation"
	validationutil "k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

func init() {
	webhookName := "validating-create-update-rolloutcontrol"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &RolloutControlCreateUpdateHandler{})
}

const (
	rolloutControlNameMaxLen = 63
)

var (
	validateRolloutControlNameMsg = "RolloutControl name must consist of alphanumeric characters or '-'"
	validateRolloutControlRegex   = regexp.MustCompile(validRolloutControlNameFmt)
	validRolloutControlNameFmt    = `^[a-zA-Z0-9\-]+$`
)

// RolloutControlCreateUpdateHandler handles RolloutControl
type RolloutControlCreateUpdateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	Client client.Client
	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *RolloutControlCreateUpdateHandler) validatingRolloutControlFn(ctx context.Context, obj *appsv1alpha1.RolloutControl) (bool, string, error) {
	allErrs := h.validateRolloutControl(obj)
	if len(allErrs) != 0 {
		return false, "", allErrs.ToAggregate()
	}
	return true, "allowed to be admitted", nil
}

func (h *RolloutControlCreateUpdateHandler) validateRolloutControl(obj *appsv1alpha1.RolloutControl) field.ErrorList {
	allErrs := genericvalidation.ValidateObjectMeta(&obj.ObjectMeta, true, validateRolloutControlName, field.NewPath("metadata"))
	allErrs = append(allErrs, h.validateRolloutControlSpec(&obj.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateRolloutControlName(name string, prefix bool) (allErrs []string) {
	if !validateRolloutControlRegex.MatchString(name) {
		allErrs = append(allErrs, validationutil.RegexError(validateRolloutControlNameMsg, validRolloutControlNameFmt, "example-com"))
	}
	if len(name) > rolloutControlNameMaxLen {
		allErrs = append(allErrs, validationutil.MaxLenError(rolloutControlNameMaxLen))
	}
	return allErrs
}

func (h *RolloutControlCreateUpdateHandler) validateRolloutControlSpec(spec *appsv1alpha1.RolloutControlSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// validate whether resource field complete
	if spec.Resource.APIVersion == "" || spec.Resource.Kind == "" || spec.Resource.NameSpace == "" || spec.Resource.Name == "" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resource"), spec.Resource, "resource should be complete"))
	}

	// validate whether find definition by resource field
	exit, err := h.findRolloutDefByControl(spec)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resource"), spec.Resource, err.Error()))
	}
	if !exit {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resource"), spec.Resource, "can't find rollout definition by "+
			spec.Resource.APIVersion+"/"+spec.Resource.Kind))
	}

	// validate whether get resource in system by .spec.resource field
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(spec.Resource.APIVersion)
	obj.SetKind(spec.Resource.Kind)
	obj.SetNamespace(spec.Resource.NameSpace)
	obj.SetName(spec.Resource.Name)
	err = h.Client.Get(context.TODO(), namespacestype.NamespacedName{
		Namespace: spec.Resource.NameSpace,
		Name:      spec.Resource.Name,
	}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("resource"), spec.Resource, "can't find resource by `.spec.resource` field"))
		} else {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("resource"), spec.Resource, err.Error()))
		}
	}

	return allErrs
}

func (h *RolloutControlCreateUpdateHandler) findRolloutDefByControl(spec *appsv1alpha1.RolloutControlSpec) (bool, error) {
	rolloutDefList := &appsv1alpha1.RolloutDefinitionList{}
	if err := h.Client.List(context.TODO(), &client.ListOptions{}, rolloutDefList); err != nil {
		return false, err
	}

	for _, rolloutDef := range rolloutDefList.Items {
		if rolloutDef.Spec.ControlResource.APIVersion == spec.Resource.APIVersion &&
			rolloutDef.Spec.ControlResource.Kind == spec.Resource.Kind {
			return true, nil
		}
	}
	return false, nil
}

var _ admission.Handler = &RolloutControlCreateUpdateHandler{}

// Handle handles admission requests.
func (h *RolloutControlCreateUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &appsv1alpha1.RolloutControl{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingRolloutControlFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

var _ inject.Client = &RolloutControlCreateUpdateHandler{}

// InjectClient injects the client into the RolloutControlCreateUpdateHandler
func (h *RolloutControlCreateUpdateHandler) InjectClient(c client.Client) error {
	h.Client = c
	return nil
}

var _ inject.Decoder = &RolloutControlCreateUpdateHandler{}

// InjectDecoder injects the decoder into the RolloutControlCreateUpdateHandler
func (h *RolloutControlCreateUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
