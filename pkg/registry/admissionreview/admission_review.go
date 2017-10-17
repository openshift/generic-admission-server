package admissionreview

import (
	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type AdmissionHookFunc func(admissionSpec admissionv1alpha1.AdmissionReviewSpec) admissionv1alpha1.AdmissionReviewStatus

type REST struct {
	hookFn AdmissionHookFunc
}

var _ rest.Creater = &REST{}

func NewREST(hookFn AdmissionHookFunc) *REST {
	return &REST{
		hookFn: hookFn,
	}
}

func (r *REST) New() runtime.Object {
	return &admissionv1alpha1.AdmissionReview{}
}

func (r *REST) Create(ctx apirequest.Context, obj runtime.Object, _ bool) (runtime.Object, error) {
	admissionReview := obj.(*admissionv1alpha1.AdmissionReview)
	admissionReview.Status = r.hookFn(admissionReview.Spec)
	return admissionReview, nil
}
