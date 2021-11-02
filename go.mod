module github.com/openshift/generic-admission-server

go 1.13

require (
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/openshift/library-go v0.0.0-20210407140145-f831e911c638
	github.com/spf13/cobra v1.1.1
	k8s.io/api v0.21.0-rc.0
	k8s.io/apimachinery v0.21.0-rc.0
	k8s.io/apiserver v0.21.0-rc.0
	k8s.io/client-go v0.21.0-rc.0
	k8s.io/component-base v0.21.0-rc.0
	k8s.io/klog/v2 v2.8.0
)
