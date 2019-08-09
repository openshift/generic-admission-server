package cmd

import (
	"flag"
	"os"
	"runtime"

	"k8s.io/klog"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	"github.com/openshift/generic-admission-server/pkg/cmd/server"
)

// AdmissionHook is what callers provide, in the mutating, the validating variant or implementing even both interfaces.
// We define it here to limit how much of the import tree callers have to deal with for this plugin. This means that
// callers need to match levels of apimachinery, api, client-go, and apiserver.
type AdmissionHook apiserver.AdmissionHook
type ValidatingAdmissionHook apiserver.ValidatingAdmissionHook
type MutatingAdmissionHook apiserver.MutatingAdmissionHook
type ConversionHook apiserver.ConversionHook

type AdmissionServerOptions struct {
	AdmissionHooks []AdmissionHook
	ConversionHooks []ConversionHook
}

func RunAdmissionServerOptions(opts AdmissionServerOptions) {
	logs.InitLogs()
	defer logs.FlushLogs()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	stopCh := genericapiserver.SetupSignalHandler()

	// done to avoid cannot use opts.AdmissionHooks (type []AdmissionHook) as type []apiserver.AdmissionHook in argument to "github.com/openshift/generic-admission-server/pkg/cmd/server".NewCommandStartAdmissionServer
	var admissionHooks []apiserver.AdmissionHook
	for i := range opts.AdmissionHooks {
		admissionHooks = append(admissionHooks, opts.AdmissionHooks[i])
	}

	// done to avoid cannot use opts.ConversionHooks (type []ConversionHook) as type []apiserver.ConversionHook in argument to "github.com/openshift/generic-admission-server/pkg/cmd/server".NewCommandStartAdmissionServer
	var conversionHooks []apiserver.ConversionHook
	for i := range opts.ConversionHooks {
		conversionHooks = append(conversionHooks, opts.ConversionHooks[i])
	}

	cmd := server.NewCommandStartAdmissionServer(os.Stdout, os.Stderr, stopCh, admissionHooks, conversionHooks)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Execute(); err != nil {
		klog.Fatal(err)
	}
}

// RunAdmissionServer runs a webhook apiserver using the given admission hooks.
// If you want to also use conversion webhooks, use the
// RunAdmissionServerOptions function instead.
func RunAdmissionServer(admissionHooks ...AdmissionHook) {
	RunAdmissionServerOptions(AdmissionServerOptions{
		AdmissionHooks:  admissionHooks,
	})
}
