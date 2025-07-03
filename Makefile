# Pull in .envrc file details, if it exists. This exists in the event you do
# not have direnv installed.
ifneq (,$(wildcard ./.envrc))
    include .envrc
    export
endif

# ====================================================================================
# Setup Project

PROJECT_NAME := controlplane-mcp-server
PROJECT_REPO := github.com/upbound/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64

# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile. If only
# "include" was used, the make command would fail and refuse
# to run a target until the include commands succeeded.
-include build/makelib/common.mk

KIND_VERSION = v0.26.0
KUBECTL_VERSION = v1.31.0
GOLANGCILINT_VERSION = 2.2.0
HELM3_VERSION = v3.16.4

KIND_CLUSTER_NAME=ctp-mcp-server

export REGISTRY_ORGS ?= xpkg.upbound.io/upbound

# ====================================================================================
# Setup Helm
USE_HELM3 = true
HELM_BASE_URL = https://charts.upbound.io
HELM_S3_BUCKET = upbound.charts
HELM_CHARTS = nothing
HELM_VALUES_TEMPLATE_SKIPPED = true

-include build/makelib/k8s_tools.mk
-include build/makelib/helm.mk

# ====================================================================================
# Setup Go

# Set a sane default so that the nprocs calculation below is less noisy on the initial
# loading of this file
NPROCS ?= 1

GO_REQUIRED_VERSION = 1.24
GOLANGCILINT_VERSION = 2.1.6
GO111MODULE = on
GO_NOCOV = true
GO_SUBDIRS = cmd
GO_LINT_DIFF_TARGET ?= HEAD~
GO_LINT_ARGS ?= --fix --new --new-from-rev=$(GO_LINT_DIFF_TARGET)
-include build/makelib/golang.mk

# ====================================================================================
# Setup Fallthrough
# We want submodules to be set up the first time `make` is run.
# We manage the build/ folder and its Makefiles as a submodule.
# The first time `make` is run, the includes of build/*.mk files will
# all fail, and this target will be run. The next time, the default as defined
# by the includes will be run instead.
fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# ====================================================================================
# Setup ctlptl

CTLPTL_VERSION := v0.8.25
CTLPTL := $(TOOLS_HOST_DIR)/ctlptl-$(CTLPTL_VERSION)

$(CTLPTL):
	@$(INFO) installing ctlptl
	@GOBIN=$(TOOLS_HOST_DIR)/tmp-ctlptl $(GO) install github.com/tilt-dev/ctlptl/cmd/ctlptl@$(CTLPTL_VERSION)
	@mv $(TOOLS_HOST_DIR)/tmp-ctlptl/ctlptl $(CTLPTL)
	@rm -fr $(TOOLS_HOST_DIR)/tmp-ctlptl
	@$(OK) installed ctlptl

# ====================================================================================
# Setup Ko

KO_VERSION := v0.17.1
export KO := $(TOOLS_HOST_DIR)/ko-$(KO_VERSION)

$(KO):
	@$(INFO) installing ko
	@GOBIN=$(TOOLS_HOST_DIR)/tmp-ko $(GO) install github.com/google/ko@$(KO_VERSION)
	@mv $(TOOLS_HOST_DIR)/tmp-ko/ko $(KO)
	@rm -fr $(TOOLS_HOST_DIR)/tmp-ko
	@$(OK) installed ko

# ====================================================================================
# Setup Tilt

TILT_VERSION ?= 0.33.6
TILT := $(TOOLS_HOST_DIR)/tilt-$(TILT_VERSION)

$(TILT):
	@$(INFO) installing tilt $(TILT_VERSION)
	@mkdir -p $(TOOLS_HOST_DIR)/tmp-tilt || $(FAIL)
	@curl -fsSL https://github.com/tilt-dev/tilt/releases/download/v$(TILT_VERSION)/tilt.$(TILT_VERSION).$(HOSTOS:darwin%=mac%).$(HOSTARCH).tar.gz | \
		tar -C $(TOOLS_HOST_DIR)/tmp-tilt -xz tilt || $(FAIL)
	@mv $(TOOLS_HOST_DIR)/tmp-tilt/tilt $@ || $(FAIL)
	@rm -fr $(TOOLS_HOST_DIR)/tmp-tilt || $(FAIL)
	@chmod +x $@ || $(FAIL)
	@$(OK) installing tilt $(TILT_VERSION)

# ====================================================================================
# Targets

# ko.publish uses ko to build and publish the following images. In addition to
# the publish step, SBOMs are produced that can be queried for by those that
# have access to the repository.
ko.publish: $(KO)
	@$(INFO) building Go artifacts using ko
	@for registry in $(REGISTRY_ORGS); do \
		$(INFO) "Publishing to $$registry"; \
			VERSION=$(VERSION) ./hack/helpers/kobuild.sh $$registry controlplane-mcp-server ./cmd/controlplane-mcp-server; \
		$(OK) "Published to $$registry"; \
	done
	@$(OK) built Go artifacts using ko

publish.artifacts: ko.publish
# run `make help` to see the targets and options

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive
# Creates a local secure registry. This is useful for publishing the local
# xpkgs to.
create.registry: $(CTLPTL)
	@$(INFO) deploying local registry
	@BUILD_REGISTRY=$(BUILD_REGISTRY) ARCH=$(ARCH) envsubst < hack/helpers/ctlptl_registry.yaml | $(CTLPTL) apply -f -
	@$(OK) deployed local registry
# Conditionally creates the local secure registry. This is useful when you are
# creating and destroying clusters but don't want to recreate the registry.
upsert.registry:
	@if [ ! $(shell ./hack/helpers/check_registry.sh) ] ; then \
		$(MAKE) create.registry; \
	fi
# Creates a kind cluster.
create.cluster: $(KIND) $(KUBECTL) upsert.registry
	@$(INFO) creating kind cluster
	@KIND=$(KIND) \
		KUBECTL=$(KUBECTL) \
		KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
		./hack/helpers/kind_with_registry.sh
	@$(OK) created kind cluster
# Simple delete of the local dev kind cluster.
delete.cluster: $(KIND)
	@$(INFO) deleting kind cluster
	@$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)
	@$(OK) deleted kind cluster
# Creates crossplane-system namespace.
create.xp.ns: $(KUBECTL)
	@$(INFO) creating crossplane-system namespace
	@$(KUBECTL) create ns crossplane-system
	@$(OK) created crossplane-system namespace
# Creates crossplane installation
create.xp: $(HELM) create.xp.ns 
	@$(INFO) creating crossplane
	@-$(HELM) repo add crossplane-stable https://charts.crossplane.io/stable
	@$(HELM) repo update
	@$(HELM) install crossplane \
		--namespace crossplane-system \
		--create-namespace crossplane-stable/crossplane \
		--wait
	@$(OK) created crossplane

cluster.up: create.cluster create.xp

cluster.down: delete.cluster

restart: cluster.down cluster.up

# ====================================================================================
# Testing
# Run go unit tests.
test.short:
	@$(GO) test -short -cover $(shell $(GO) list ./... | grep -v test\/e2e)

# ====================================================================================
# Local dev

# Run the MCP Inspector
# The mcp server uses a KUBECONFIG for auth to the k8s api-server, but does not
# protect the individual endpoints.
inspector:
	@DANGEROUSLY_OMIT_AUTH=true npx @modelcontextprotocol/inspector

# Run the ControlPlane MCP Server locally
runmcp:
	@$(GO) run ./cmd/controlplane-mcp-server/...