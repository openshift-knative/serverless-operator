# IMAGE_NAME needs to be passed explicitly, e.g.
# make IMAGE_NAME=openshift-knative/must-gather
ifndef IMAGE_NAME
$(error IMAGE_NAME is not set.)
endif

IMAGE_REGISTRY ?= quay.io
IMAGE_TAG ?= latest

build: docker-build docker-push

docker-build:
	docker build -t ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} .

docker-push:
	docker push ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}

.PHONY: build docker-build docker-push
