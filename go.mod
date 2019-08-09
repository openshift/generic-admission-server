module github.com/openshift/generic-admission-server

go 1.12

require (
	github.com/evanphx/json-patch v4.2.0+incompatible // indirect
	github.com/json-iterator/go v1.1.7 // indirect
	github.com/spf13/cobra v0.0.4
	golang.org/x/net v0.0.0-20190311183353-d8887717615a // indirect
	k8s.io/api v0.0.0-20190805141119-fdd30b57c827
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/apiserver v0.0.0-20190805142138-368b2058237c
	k8s.io/client-go v0.0.0-20190805141520-2fe0317bcee0
	k8s.io/component-base v0.0.0-20190805141645-3a5e5ac800ae
	k8s.io/klog v0.3.1
)
