# generic-admission-server
A library for writing admission webhooks based on k8s.io/apiserver


```go
import "github.com/openshift/generic-admission-server/pkg/cmd"

func main() {
	cmd.RunAdmissionServer(&admissionHook{})
}

// where to host it
func (a *admissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {}

// your business logic
func (a *admissionHook) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {}

// any special initialization goes here
func (a *admissionHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {}
```

## Examples of Projects that use Openshift Generic Admission Server

Here are a selection of webhooks which use the Openshift Generic Admission Server:

* [Openshift Kubernetes Namespace Reservation](https://github.com/openshift/kubernetes-namespace-reservation): An admission webhook that prevents the creation of specified namespaces.
* [Quack](https://github.com/pusher/quack): In-Cluster templating for Kubernetes manifests.
* [Cert-Manager Validating Webhook](https://docs.cert-manager.io/en/latest/getting-started/webhook.html): Allows cert-manager to validate that Issuer, ClusterIssuer and Certificate resources that are submitted to the apiserver are syntactically valid.
* [Anchore Image Validator](https://github.com/banzaicloud/anchore-image-validator): Lets you automatically detect or block security issues just before a Kubernetes pod starts.
