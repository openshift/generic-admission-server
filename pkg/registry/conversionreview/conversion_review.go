package conversionreview

import (
	"context"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

type ConversionHookFunc func(conversionSpec *apiextensionsv1beta1.ConversionRequest) *apiextensionsv1beta1.ConversionResponse

type REST struct {
	hookFn ConversionHookFunc
}

var _ rest.Creater = &REST{}
var _ rest.Scoper = &REST{}
var _ rest.GroupVersionKindProvider = &REST{}

func NewREST(hookFn ConversionHookFunc) *REST {
	return &REST{
		hookFn: hookFn,
	}
}

func (r *REST) New() runtime.Object {
	return &apiextensionsv1beta1.ConversionReview{}
}

func (r *REST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return apiextensionsv1beta1.SchemeGroupVersion.WithKind("ConversionReview")
}

func (r *REST) NamespaceScoped() bool {
	return false
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	conversionReview := obj.(*apiextensionsv1beta1.ConversionReview)
	conversionReview.Response = r.hookFn(conversionReview.Request)
	return conversionReview, nil
}
