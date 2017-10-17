package cmd

import (
	"flag"
	"os"
	"runtime"

	"github.com/golang/glog"

	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/rest"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	"github.com/openshift/generic-admission-server/pkg/cmd/server"
)

// AdmissionHook is what callers provide.  We define it here to limit how much of the import tree
// callers have to deal with for this plugin.  This means that callers need to match levels of
// apimachinery, api, client-go, and apiserver.
type AdmissionHook interface {
	// Resource is the resource to use for hosting your admission webhook
	Resource() (plural schema.GroupVersionResource, singular string)

	// Admit is called to decide whether to accept the admission request.
	Admit(admissionSpec admissionv1alpha1.AdmissionReviewSpec) admissionv1alpha1.AdmissionReviewStatus

	// Initialize is called as a post-start hook
	Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error
}

func RunAdmission(admissionHooks ...AdmissionHook) {
	logs.InitLogs()
	defer logs.FlushLogs()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	stopCh := genericapiserver.SetupSignalHandler()

	// done to avoid cannot use admissionHooks (type []AdmissionHook) as type []apiserver.AdmissionHook in argument to "github.com/openshift/kubernetes-namespace-reservation/pkg/genericadmissionserver/cmd/server".NewCommandStartNamespaceReservationServer
	castSlice := []apiserver.AdmissionHook{}
	for i := range admissionHooks {
		castSlice = append(castSlice, admissionHooks[i])
	}
	cmd := server.NewCommandStartNamespaceReservationServer(os.Stdout, os.Stderr, stopCh, castSlice...)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Execute(); err != nil {
		glog.Fatal(err)
	}
}
