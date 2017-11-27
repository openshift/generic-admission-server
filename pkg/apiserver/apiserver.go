package apiserver

import (
	"fmt"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apimachinery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	restclient "k8s.io/client-go/rest"

	"github.com/openshift/generic-admission-server/pkg/registry/admissionreview"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

type AdmissionHook interface {
	// Resource is the resource to use for hosting your admission webhook
	Resource() (plural schema.GroupVersionResource, singular string)

	// Validate is called to decide whether to accept the admission request.
	Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse

	// Initialize is called as a post-start hook
	Initialize(kubeClientConfig *restclient.Config, stopCh <-chan struct{}) error
}

func init() {
	admissionv1beta1.AddToScheme(Scheme)

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

type ExtraConfig struct {
	AdmissionHooks []AdmissionHook
}

// NamespaceReservationServer contains state for a Kubernetes cluster master/api server.
type NamespaceReservationServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() CompletedConfig {
	completedCfg := completedConfig{
		c.GenericConfig.Complete(),
		&c.ExtraConfig,
	}

	completedCfg.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&completedCfg}
}

// New returns a new instance of NamespaceReservationServer from the given config.
func (c completedConfig) New() (*NamespaceReservationServer, error) {
	genericServer, err := c.GenericConfig.New("kubernetes-namespace-reservation", genericapiserver.EmptyDelegate) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	s := &NamespaceReservationServer{
		GenericAPIServer: genericServer,
	}

	inClusterConfig, err := restclient.InClusterConfig()
	if err != nil {
		return nil, err
	}

	for _, versionMap := range admissionHooksByGroupThenVersion(c.ExtraConfig.AdmissionHooks...) {
		accessor := meta.NewAccessor()
		versionInterfaces := &meta.VersionInterfaces{
			ObjectConvertor:  Scheme,
			MetadataAccessor: accessor,
		}
		interfacesFor := func(version schema.GroupVersion) (*meta.VersionInterfaces, error) {
			if version != admissionv1beta1.SchemeGroupVersion {
				return nil, fmt.Errorf("unexpected version %v", version)
			}
			return versionInterfaces, nil
		}
		restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{admissionv1beta1.SchemeGroupVersion}, interfacesFor)
		// TODO we're going to need a later k8s.io/apiserver so that we can get discovery to list a different group version for
		// our endpoint which we'll use to back some custom storage which will consume the AdmissionReview type and give back the correct response
		apiGroupInfo := genericapiserver.APIGroupInfo{
			GroupMeta: apimachinery.GroupMeta{
				// filled in later
				//GroupVersion:  admissionVersion,
				//GroupVersions: []schema.GroupVersion{admissionVersion},

				SelfLinker:    runtime.SelfLinker(accessor),
				RESTMapper:    restMapper,
				InterfacesFor: interfacesFor,
				InterfacesByVersion: map[schema.GroupVersion]*meta.VersionInterfaces{
					admissionv1beta1.SchemeGroupVersion: versionInterfaces,
				},
			},
			VersionedResourcesStorageMap: map[string]map[string]rest.Storage{},
			// TODO unhardcode this.  It was hardcoded before, but we need to re-evaluate
			OptionsExternalVersion: &schema.GroupVersion{Version: "v1"},
			Scheme:                 Scheme,
			ParameterCodec:         metav1.ParameterCodec,
			NegotiatedSerializer:   Codecs,
		}

		for _, admissionHooks := range versionMap {
			for i := range admissionHooks {
				admissionHook := admissionHooks[i]
				admissionResource, singularResourceType := admissionHook.Resource()
				admissionVersion := admissionResource.GroupVersion()

				restMapper.AddSpecific(
					admissionv1beta1.SchemeGroupVersion.WithKind("AdmissionReview"),
					admissionResource,
					admissionVersion.WithResource(singularResourceType),
					meta.RESTScopeRoot)

				// just overwrite the groupversion with a random one.  We don't really care or know.
				apiGroupInfo.GroupMeta.GroupVersions = append(apiGroupInfo.GroupMeta.GroupVersions, admissionVersion)

				admissionReview := admissionreview.NewREST(admissionHook.Validate)
				v1alpha1storage := map[string]rest.Storage{
					admissionResource.Resource: admissionReview,
				}
				apiGroupInfo.VersionedResourcesStorageMap[admissionVersion.Version] = v1alpha1storage

				s.GenericAPIServer.AddPostStartHookOrDie(
					fmt.Sprintf("%s.%s.%s-init", admissionResource.Resource, admissionResource.Version, admissionResource.Group),
					func(context genericapiserver.PostStartHookContext) error {
						return admissionHook.Initialize(inClusterConfig, context.StopCh)
					},
				)
			}
		}

		// just prefer the first one in the list for consistency
		apiGroupInfo.GroupMeta.GroupVersion = apiGroupInfo.GroupMeta.GroupVersions[0]

		if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func admissionHooksByGroupThenVersion(admissionHooks ...AdmissionHook) map[string]map[string][]AdmissionHook {
	ret := map[string]map[string][]AdmissionHook{}

	for i := range admissionHooks {
		gvr, _ := admissionHooks[i].Resource()

		group, ok := ret[gvr.Group]
		if !ok {
			group = map[string][]AdmissionHook{}
			ret[gvr.Group] = group
		}

		group[gvr.Version] = append(group[gvr.Version], admissionHooks[i])
	}

	return ret
}
