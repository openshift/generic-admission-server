package main

import (
	"github.com/openshift/generic-admission-server/pkg/cmd"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func main() {
	cmd.RunAdmissionServer(&admissionHook{})
}

type admissionHook struct {
}

// where to host it
func (a *admissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
		Group:    "samples.admission.openshift.io",
		Version:  "v1",
		Resource: "dumprequests",
	}, "dumprequest"
}

// your business logic
func (a *admissionHook) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	output, err := json.Marshal(admissionSpec)
	if err != nil {
		utilruntime.HandleError(err)
		return &admissionv1beta1.AdmissionResponse{
			UID:     admissionSpec.UID,
			Allowed: true,
		}
	}
	klog.Infof("Received\n%v", string(output))
	return &admissionv1beta1.AdmissionResponse{
		UID:     admissionSpec.UID,
		Allowed: true,
	}
}

// any special initialization goes here
func (a *admissionHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	return nil
}
