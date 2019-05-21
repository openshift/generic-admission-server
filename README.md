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

## Why use this library?

This library helps you to write secure [Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).
It uses TLS authentication and authorization mechanisms which are built into the [Kubernetes aggregated API server](https://github.com/kubernetes/apiserver) library,
which means that your webhooks are secure by default.

Using this library allows you to avoid the complication of creating and maintaining a client key and certificate for each webhook server;
you only need to maintain a server key and certificate for each webhook server.
And by using this library your webhook will also perform authorization which uses Kubernetes' own `SubjectAccessReview` and `RBAC` mechanisms.

## Deployment

Deploy your webhook as an [aggregated API server](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/).
This provides one or more new Kubernetes API endpoints which are served by the Kubernetes API server its self.
E.g. `/apis/admission.core.example.com/v1/flunders`
Ensure that these endpoints are accessible before continuing.

Then [configure admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#configure-admission-webhooks-on-the-fly) which target this new endpoint on the Kubernetes API server.
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

## Architecture

Kubernetes API servers connect to webhook servers using TLS encrypted HTTPS connections.
In a production environment, the [Kubernetes API servers should also be configured to authenticate themselves to webhook servers](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers),
and your webhook servers should verify the authenticity of the requests they receive from Kubernetes API servers.
But this is (currently) rather complicated to maintain because you have to provide a `kubeConfig` file containing the client authentication configuration for each webhook server.

An alternative approach, used by this library, is to deploy the webhook server as a [Kubernetes aggregated API server](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/).
The advantage of this approach is that the [mechanism for establishing mutual authentication between the Kubernetes API server and aggregate API servers is more mature and easier to maintain](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/#authentication-flow).
In this mechanism, Kubernetes takes care of generating the client authentication credentials that it uses when connecting to aggregate API servers.
And the aggregate API server then reads these client credentials from a standardized `ConfigMap` at `kube-system/extension-apiserver-authentication`.
The [Kubernetes API server library](https://github.com/kubernetes/apiserver) takes care of all this for you.

Additional security is provided because the [webhook aggregate API server authorizes the request](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/#extension-apiserver-authorizes-the-request).
The webhook aggregate API server will receive the username and group of the user or service account that made the request that triggered the web hook.
And it will check, using a `SubjectAccessReview`, that that original user has permission to interact with this webhook API.

## FAQ

### Why can't I write a simple HTTP webhook server?

Admission webhooks have tremendous power over what can and cannot be created in the API.
They can see, validate, and in some cases mutate every object in the cluster,
so it is vital that the API server can verify that it is connecting to an authentic webhook server.
And it is also vital that a webhook server can verify that it is receiving requests from an authentic Kubernetes API server.
Kubernetes will eventually deprecate and remove all unencrypted HTTP APIs.

### OK, but how am I supposed to manage all the TLS certificates for my web hooks?

For testing purposes, you can create a private key and a self-signed certificate using `openssl` or `cfssl`.

In production, you must implement a process for rotating the certificates.
For example:
* [OpenShift Service CA Operator](https://github.com/openshift/service-ca-operator): Controller to mint and manage serving certificates for Kubernetes services.
* [cert-manager](https://docs.cert-manager.io/en/latest/tasks/issuers/setup-ca.html): A controller for automatically provisioning and managing TLS certificates in Kubernetes.

## Examples of Projects that use Openshift Generic Admission Server

Here are a selection of webhooks which use the Openshift Generic Admission Server:

* [Openshift Kubernetes Namespace Reservation](https://github.com/openshift/kubernetes-namespace-reservation): An admission webhook that prevents the creation of specified namespaces.
* [Quack](https://github.com/pusher/quack): In-Cluster templating for Kubernetes manifests.
* [Cert-Manager Validating Webhook](https://docs.cert-manager.io/en/latest/getting-started/webhook.html): Allows cert-manager to validate that Issuer, ClusterIssuer and Certificate resources that are submitted to the apiserver are syntactically valid.
* [Anchore Image Validator](https://github.com/banzaicloud/anchore-image-validator): Lets you automatically detect or block security issues just before a Kubernetes pod starts.
