GO ?= $(shell which go)
OS ?= $(shell $(GO) env GOOS)
ARCH ?= $(shell $(GO) env GOARCH)

IMAGE_NAME := "localdns"
IMAGE_TAG := "latest"

OUT := $(shell pwd)/_out

KUBE_VERSION=1.35.0

$(shell mkdir -p "$(OUT)")

test:
	KUBEBUILDER_ASSETS=$$(go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use $(KUBE_VERSION) -p path) && \
	TEST_ASSET_KUBE_APISERVER=$$KUBEBUILDER_ASSETS/kube-apiserver \
	TEST_ASSET_ETCD=$$KUBEBUILDER_ASSETS/etcd \
	TEST_ASSET_KUBECTL=$$KUBEBUILDER_ASSETS/kubectl \
	$(GO) test -v .

clean:
	rm -rf $(OUT)

build:
	docker build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: rendered-manifest.yaml
rendered-manifest.yaml:
	helm template \
	    --name localdns \
            --set image.repository=$(IMAGE_NAME) \
            --set image.tag=$(IMAGE_TAG) \
            deploy/localdns > "$(OUT)/rendered-manifest.yaml"
