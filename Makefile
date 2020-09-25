all: build
.PHONY: all

openapi: install-openapigen
	openapi-gen --logtostderr -i k8s.io/apimachinery/pkg/apis/meta/v1 -p github.com/openshift/generic-admission-server/pkg/generated/openapi/ -O zz_generated.openapi -h hack/boilerplate.go.txt -r /dev/null

install-openapigen:
	go install k8s.io/kube-openapi/cmd/openapi-gen

build:
	GO111MODULE=on GOPROXY=https://proxy.golang.org go build -o _output/bin/generic-admission-server github.com/openshift/generic-admission-server/pkg/cmd
.PHONY: build

clean:
	rm -rf _output
.PHONY: clean

update-deps:
	hack/update-deps.sh
.PHONY: generate
