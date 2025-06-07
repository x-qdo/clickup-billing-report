# Makefile for lambda-opsbot

# Configuration
ECR_URL := 898445925860.dkr.ecr.eu-west-1.amazonaws.com
ECR_REPO := $(ECR_URL)/clickup-report
ECR_REGION := eu-west-1
VERSION ?= latest
PLATFORM := linux/arm64

# Docker image names
HTTP_IMAGE := $(ECR_REPO):$(VERSION)-http

.PHONY: help build build-http build-sqs push login clean

help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: build-http

build-http: ## Build the HTTP Docker image
	docker buildx build --provenance=false --platform $(PLATFORM) -t $(HTTP_IMAGE) .

push: ## Push both Docker images to ECR
	docker push $(HTTP_IMAGE)

login: ## Log in to ECR
	aws ecr get-login-password --region $(ECR_REGION) | docker login --username AWS --password-stdin $(ECR_URL)

clean: ## Remove local Docker images
	docker rmi $(HTTP_IMAGE) || true

# Usage examples:
# make build VERSION=1.0.0
# make push VERSION=1.0.0
# make build push VERSION=1.0.0
