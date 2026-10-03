.DEFAULT_GOAL := help

# Container image settings
REGISTRY       ?= quay.io/rh-ai-community-plugins
FRONTEND_IMAGE  ?= gpu-booking
BFF_IMAGE       ?= gpu-booking-bff
CHART_NAME      ?= gpu-booking-chart  # used by help display only; chart-package/chart-push use chart/ directly
IMAGE_TAG      ?= latest
BUILDER        ?= podman

# ──────────────────────────────────────────────
# Install
# ──────────────────────────────────────────────

.PHONY: install

install: ## Install frontend dependencies
	npm ci

# ──────────────────────────────────────────────
# Lint
# ──────────────────────────────────────────────

.PHONY: lint lint-chart

lint: ## Lint source code (frontend)
	npm run lint

lint-chart: ## Lint the Helm chart
	helm lint chart/

# ──────────────────────────────────────────────
# Typecheck
# ──────────────────────────────────────────────

.PHONY: typecheck

typecheck: ## TypeScript type checking (frontend)
	npm run typecheck

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────

.PHONY: test test-frontend test-go

test: test-frontend test-go ## Run all tests (frontend + Go)

test-frontend: ## Run frontend tests (vitest)
	npm test

test-go: ## Run Go backend tests
	go test ./pkg/...

# ──────────────────────────────────────────────
# Validate (typecheck + lint + test)
# ──────────────────────────────────────────────

.PHONY: validate

validate: typecheck lint test lint-chart ## Full validation: typecheck + lint + tests + chart lint

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

.PHONY: build build-dev build-go

build: ## Production frontend build to dist/
	npm run build

build-dev: ## Development frontend build (webpack dev mode)
	npm run build:dev

build-go: ## Build the Go backend binary to bin/
	CGO_ENABLED=1 go build -o bin/backend ./cmd/backend/

# ──────────────────────────────────────────────
# Dev servers
# ──────────────────────────────────────────────

.PHONY: dev dev-bff

dev: ## Start frontend dev server (port 9500)
	npm run start:dev

dev-bff: ## Run the Go backend locally on :3000 (dev auth, plain HTTP)
	DEV_MODE=true PORT=3000 go run ./cmd/backend/

# ──────────────────────────────────────────────
# Container images
# ──────────────────────────────────────────────

.PHONY: image image-frontend image-bff image-push image-push-frontend image-push-bff

image: image-frontend image-bff ## Build container images (frontend + backend)

image-frontend:
	$(BUILDER) build -t $(REGISTRY)/$(FRONTEND_IMAGE):$(IMAGE_TAG) -f Containerfile .

image-bff:
	$(BUILDER) build -t $(REGISTRY)/$(BFF_IMAGE):$(IMAGE_TAG) -f Containerfile.bff .

image-push: image-push-frontend image-push-bff ## Build and push container images (frontend + backend)

image-push-frontend: image-frontend ## Build and push the frontend container image
	$(BUILDER) push $(REGISTRY)/$(FRONTEND_IMAGE):$(IMAGE_TAG)

image-push-bff: image-bff ## Build and push the backend container image
	$(BUILDER) push $(REGISTRY)/$(BFF_IMAGE):$(IMAGE_TAG)

# ──────────────────────────────────────────────
# Helm chart
# ──────────────────────────────────────────────

.PHONY: chart-package chart-push

chart-package: ## Package Helm chart into a .tgz archive
	helm package chart/

chart-push: ## Package and push Helm chart to OCI registry (requires Helm 3.8+)
	$(eval CHART_TGZ := $(shell helm package chart/ | awk '{print $$NF}'))
	helm push $(CHART_TGZ) oci://$(REGISTRY)
	@rm -f $(CHART_TGZ)

# ──────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────

.PHONY: clean

clean: ## Remove build artifacts
	rm -rf dist/ bin/ coverage/ coverage-go/

# ──────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────

.PHONY: help

help: ## Show this help
	@printf "\n\033[1mTargets:\033[0m\n"
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@printf "\n\033[1mVariables:\033[0m\n"
	@printf "  \033[33m%-20s\033[0m %s (default: %s)\n" "REGISTRY"       "Container image registry"       "$(REGISTRY)"
	@printf "  \033[33m%-20s\033[0m %s (default: %s)\n" "FRONTEND_IMAGE" "Frontend image name"            "$(FRONTEND_IMAGE)"
	@printf "  \033[33m%-20s\033[0m %s (default: %s)\n" "BFF_IMAGE"      "Backend (Go) image name"        "$(BFF_IMAGE)"
	@printf "  \033[33m%-20s\033[0m %s (default: %s)\n" "CHART_NAME"     "Helm chart name"                "$(CHART_NAME)"
	@printf "  \033[33m%-20s\033[0m %s (default: %s)\n" "IMAGE_TAG"      "Tag for image / image-push"     "$(IMAGE_TAG)"
	@printf "  \033[33m%-20s\033[0m %s (default: %s)\n" "BUILDER"        "Container build tool"           "$(BUILDER)"
	@printf "\n\033[1mExamples:\033[0m\n"
	@printf "  make validate                          # typecheck + lint + tests + chart lint\n"
	@printf "  make image                            # build both container images\n"
	@printf "  make image-push IMAGE_TAG=0.1.0       # build and push with explicit tag\n"
	@printf "  make chart-push                       # package and push Helm chart to OCI registry\n"
	@printf "\n"
