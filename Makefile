# Makefile for clickup-report

# Configuration
ECR_URL := 898445925860.dkr.ecr.eu-west-1.amazonaws.com
ECR_REPO := $(ECR_URL)/clickup-report
ECR_REGION := eu-west-1
VERSION ?= latest
PLATFORM := linux/arm64

# Docker image names
HTTP_IMAGE := $(ECR_REPO):$(VERSION)-http
WORKER_IMAGE := $(ECR_REPO):$(VERSION)-worker

.PHONY: help build build-http build-worker push push-http push-worker login clean

help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: build-http build-worker ## Build all Docker images

build-http: ## Build the HTTP Docker image
	docker buildx build --provenance=false --platform $(PLATFORM) --build-arg BUILD_TARGET=http -t $(HTTP_IMAGE) .

build-worker: ## Build the Worker Docker image
	docker buildx build --provenance=false --platform $(PLATFORM) --build-arg BUILD_TARGET=worker -t $(WORKER_IMAGE) .

push: push-http push-worker ## Push all Docker images to ECR

push-http: ## Push HTTP Docker image to ECR
	docker push $(HTTP_IMAGE)

push-worker: ## Push Worker Docker image to ECR
	docker push $(WORKER_IMAGE)

login: ## Log in to ECR
	aws ecr get-login-password --region $(ECR_REGION) | docker login --username AWS --password-stdin $(ECR_URL)

clean: ## Remove local Docker images
	docker rmi $(HTTP_IMAGE) || true
	docker rmi $(WORKER_IMAGE) || true

# Usage examples:
# make build VERSION=1.0.0
# make push VERSION=1.0.0
# make build push VERSION=1.0.0
