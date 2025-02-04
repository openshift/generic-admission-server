package apiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"testing"

	"net/http"
	"net/http/httptest"

	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	kubeopenapi "k8s.io/kube-openapi/pkg/common"
)

const (
	validatorPath = "/apis/admission.openshift.io/v1/testvalidators"
	mutatorPath   = "/apis/admission.openshift.io/v1/testmutators"
)

type testWebhook struct {
}

// Initialize is called by generic-admission-server on startup to setup initialization that webhook needs.
func (a *testWebhook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	// do nothing
	return nil
}

func (a *testWebhook) MutatingResource() (schema.GroupVersionResource, string) {
	return schema.GroupVersionResource{
			Group:    "admission.openshift.io",
			Version:  "v1",
			Resource: "testmutators",
		},
		"testmutators"
}

func (a *testWebhook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.openshift.io",
			Version:  "v1",
			Resource: "testvalidators",
		},
		"testvalidators"
}

type testWebhookV1Beta1 struct {
	testWebhook
}

func (a *testWebhookV1Beta1) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{Allowed: true}
}

func (a *testWebhookV1Beta1) Admit(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{Allowed: true, Patch: []byte("{}")}
}

type testWebhookV1 struct {
	testWebhook
}

func (a *testWebhookV1) Validate(admissionSpec *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{Allowed: true}
}

func (a *testWebhookV1) Admit(admissionSpec *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{Allowed: true, Patch: []byte("{}")}
}

func TestV1Beta1Webhook(t *testing.T) {
	testHook := &testWebhookV1Beta1{}
	server := newTestServer(t, testHook)
	defer server.Close()

	reviewRequest := &admissionv1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1beta1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1beta1.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Kind: "TestKind"},
		},
	}
	payload, _ := json.Marshal(reviewRequest)

	cases := []struct {
		name             string
		path             string
		validateResponse func(t *testing.T, response *admissionv1beta1.AdmissionResponse)
	}{
		{
			name: "test validator",
			path: validatorPath,
			validateResponse: func(t *testing.T, response *admissionv1beta1.AdmissionResponse) {
				if response == nil {
					t.Errorf("expect review response but get nil")
				}
				if response.Allowed != true {
					t.Errorf("expect validation is allowed")
				}
			},
		},
		{
			name: "test mutator",
			path: mutatorPath,
			validateResponse: func(t *testing.T, response *admissionv1beta1.AdmissionResponse) {
				if response == nil {
					t.Errorf("expect review response but get nil")
				}
				if string(response.Patch) != "{}" {
					t.Errorf("unexpected mutator response; %v", response)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			url := fmt.Sprintf("%s%s", server.URL, c.path)
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
			if err != nil {
				t.Errorf("unexpected error when calling webhook, but got %v", err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("unexpected error reading body at url %q: %v", url, err)
			}

			reviewResponse := &admissionv1beta1.AdmissionReview{}
			err = json.Unmarshal(body, reviewResponse)
			if err != nil {
				t.Errorf("unexpected error parsing json body at path %q: %v", url, err)
			}

			c.validateResponse(t, reviewResponse.Response)
		})
	}
}

func TestV1Webhook(t *testing.T) {
	testHook := &testWebhookV1{}
	server := newTestServer(t, testHook)
	defer server.Close()

	reviewRequest := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Kind: "TestKind"},
		},
	}
	payload, _ := json.Marshal(reviewRequest)

	cases := []struct {
		name             string
		path             string
		validateResponse func(t *testing.T, response *admissionv1.AdmissionResponse)
	}{
		{
			name: "test validator",
			path: validatorPath,
			validateResponse: func(t *testing.T, response *admissionv1.AdmissionResponse) {
				if response == nil {
					t.Errorf("expect review response but get nil")
				}
				if response.Allowed != true {
					t.Errorf("expect validation is allowed")
				}
			},
		},
		{
			name: "test mutator",
			path: mutatorPath,
			validateResponse: func(t *testing.T, response *admissionv1.AdmissionResponse) {
				if response == nil {
					t.Errorf("expect review response but get nil")
				}
				if string(response.Patch) != "{}" {
					t.Errorf("unexpected mutator response; %v", response)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			url := fmt.Sprintf("%s%s", server.URL, c.path)
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
			if err != nil {
				t.Errorf("unexpected error when calling webhook, but got %v", err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("unexpected error reading body at url %q: %v", url, err)
			}

			fmt.Printf("body is %v\n", string(body))

			reviewResponse := &admissionv1.AdmissionReview{}
			err = json.Unmarshal(body, reviewResponse)
			if err != nil {
				t.Errorf("unexpected error parsing json body at path %q: %v", url, err)
			}

			c.validateResponse(t, reviewResponse.Response)
		})
	}
}

func newTestServer(t *testing.T, webhook AdmissionHook) *httptest.Server {
	serverConfig := genericapiserver.NewRecommendedConfig(Codecs)
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(testGetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(runtime.NewScheme()))
	serverConfig.OpenAPIConfig.Info.Version = "unversioned"
	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(testGetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(runtime.NewScheme()))
	serverConfig.OpenAPIV3Config.Info.Version = "unversioned"
	serverConfig.ExternalAddress = "192.168.10.4:443"
	serverConfig.PublicAddress = net.ParseIP("192.168.10.4")
	serverConfig.LegacyAPIGroupPrefixes = sets.NewString("/api")
	serverConfig.LoopbackClientConfig = &restclient.Config{}

	config := &Config{
		GenericConfig: serverConfig,
		ExtraConfig: ExtraConfig{
			[]AdmissionHook{webhook},
		},
		RestConfig: &restclient.Config{},
	}

	addmissionServer, err := config.Complete().New()
	if err != nil {
		t.Errorf("unexpected error building server: %v", err)
	}
	server := httptest.NewServer(addmissionServer.GenericAPIServer.Handler)
	return server
}

func testGetOpenAPIDefinitions(_ kubeopenapi.ReferenceCallback) map[string]kubeopenapi.OpenAPIDefinition {
	return map[string]kubeopenapi.OpenAPIDefinition{
		"k8s.io/api/admission/v1.AdmissionReview":      {},
		"k8s.io/api/admission/v1beta1.AdmissionReview": {},
	}
}
