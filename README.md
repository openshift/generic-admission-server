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


## Architecture

This library helps you to write secure [Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).
It uses TLS authentication functions which are built into the [Kubernetes aggregated API server](https://github.com/kubernetes/apiserver) library,
which means that your webhooks are secure by default.

A `generic-admission-server` based webhook server is first deployed as an [aggregated API server](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/).
This provides one or more new Kubernetes API endpoints which are served by the Kubernetes API server its self.
E.g. `/apis/admission.core.example.com/v1/flunders`

You then [configure admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#configure-admission-webhooks-on-the-fly) which target this new endpoint on the Kubernetes API server.
E.g.

```
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: example-webhook
webhooks:
  - name: admission.core.example.com
    rules:
      - apiGroups:
          - "core.example.com"
        apiVersions:
          - v1
        operations:
          - CREATE
          - UPDATE
        resources:
          - flunders
    failurePolicy: Fail
    clientConfig:
      service:
        name: kubernetes
        namespace: default
        path: /apis/admission.core.example.com/v1/flunders
      caBundle: $CA_BUNDLE
```

In this way, the [MutatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook) or [ValidatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook) admission controllers, running in the Kubernetes API server process, are looping back to the main Kubernetes API service.


## FAQ

### Why can't I write a simple HTTP webhook server?

Admission webhooks have tremendous power over what can and cannot be created in the API.
They can see, validate, and in some cases mutate every object in the cluster,
so it is vital that the API server can verify that it is connecting to an authentic webhook server.
And it is also vital that a webhook server can verify that it is receiving requests from an authentic Kubernetes API server.
Kubernetes will eventually deprecate and remove all unencrypted HTTP APIs.


## Examples of Projects that use Openshift Generic Admission Server

Here are a selection of webhooks which use the Openshift Generic Admission Server:

* [Openshift Kubernetes Namespace Reservation](https://github.com/openshift/kubernetes-namespace-reservation): An admission webhook that prevents the creation of specified namespaces.
* [Quack](https://github.com/pusher/quack): In-Cluster templating for Kubernetes manifests.
* [Cert-Manager Validating Webhook](https://docs.cert-manager.io/en/latest/getting-started/webhook.html): Allows cert-manager to validate that Issuer, ClusterIssuer and Certificate resources that are submitted to the apiserver are syntactically valid.
* [Anchore Image Validator](https://github.com/banzaicloud/anchore-image-validator): Lets you automatically detect or block security issues just before a Kubernetes pod starts.
