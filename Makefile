# Modified for keepalived-cloud-provider
#
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

all: build test verify

# Some env vars that devs might find useful:
#  GOFLAGS      : extra "go build" flags to use - e.g. -v   (for verbose)
#  NO_DOCKER=1  : execute each step natively, not in a Docker container
#  TEST_DIRS=   : only run the unit tests from the specified dirs
#  UNIT_TESTS=  : only run the unit tests matching the specified regexp

# Define some constants
#######################
ROOT           = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BINDIR        ?= bin
BUILD_DIR     ?= build
KCP_PKG         = github.com/munnerz/keepalived-cloud-provider
TOP_SRC_DIRS   = keepalivedcp
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
VERSION       ?= $(shell git describe --always --tags --abbrev=7 --dirty)
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
NEWEST_GO_FILE = $(shell find $(SRC_DIRS) -name \*.go -exec $(STAT) {} \; \
                   | sort -r | head -n 1 | sed "s/.* //")
GO_VERSION     = 1.8.0

PLATFORM?=linux
ARCH?=amd64

GO_BUILD       = env CGO_ENABLED=0 GOOS=$(PLATFORM) GOARCH=$(ARCH) go build -i $(GOFLAGS) \
                   -ldflags "-X main.version=$(VERSION)"
BASE_PATH      = $(ROOT:/src/github.com/munnerz/keepalived-cloud-provider/=)
export GOPATH  = $(BASE_PATH):$(ROOT)/vendor

MUTABLE_TAG                      ?= canary
CONTROLLER_IMAGE          = $(REGISTRY)keepalived-cloud-provider:$(VERSION)
CONTROLLER_MUTABLE_IMAGE  = $(REGISTRY)keepalived-cloud-provider:$(MUTABLE_TAG)

ifdef UNIT_TESTS
	UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

ifdef NO_DOCKER
	DOCKER_CMD =
	kcpBuildImageTarget =
else
	# Mount .pkg as pkg so that we save our cached "go build" output files
	DOCKER_CMD = docker run --rm -v $(PWD):/go/src/$(KCP_PKG) \
	  -v $(PWD)/.pkg:/go/pkg kcpbuildimage
	kcpBuildImageTarget = .kcpBuildImage
endif

NON_VENDOR_DIRS = $(shell $(DOCKER_CMD) go list $(KCP_PKG)/... | grep -v /vendor/)

# Some prereq stuff
###################
.init: $(kcpBuildImageTarget)

.kcpBuildImage: build/build-image/Dockerfile
	sed "s/GO_VERSION/$(GO_VERSION)/g" < build/build-image/Dockerfile | \
	  docker build -t kcpbuildimage -
	touch $@

# This section builds the output binaries.
#########################################################################
build: .init \
       $(BINDIR)/keepalived-cloud-provider

keepalived-cloud-provider: $(BINDIR)/keepalived-cloud-provider
$(BINDIR)/keepalived-cloud-provider: .init
	$(DOCKER_CMD) $(GO_BUILD) -o $@ $(KCP_PKG)


# Util targets
##############
.PHONY: verify
verify: .init
	@echo Running gofmt:
	@$(DOCKER_CMD) gofmt -l -s $(TOP_SRC_DIRS) > .out 2>&1 || true
	@bash -c '[ "`cat .out`" == "" ] || \
	  (echo -e "\n*** Please 'gofmt' the following:" ; cat .out ; echo ; false)'
	@rm .out
	@#
	@echo Running golint and go vet:
	@# Exclude the generated (zz) files for now, as well as defaults.go (it
	@# observes conventions from upstream that will not pass lint checks).
	@$(DOCKER_CMD) sh -c \
	  'for i in $$(find $(TOP_SRC_DIRS) -name \\*.go); \
	  do \
	   golint --set_exit_status $$i || exit 1; \
	  done'
	@#
	$(DOCKER_CMD) go vet $(NON_VENDOR_DIRS)

format: .init
	$(DOCKER_CMD) gofmt -w -s $(TOP_SRC_DIRS)

test: .init build test-unit

test-unit: .init build
	@echo Running tests:
	$(DOCKER_CMD) go test $(UNIT_TEST_FLAGS) \
	  $(addprefix $(KCP_PKG)/,$(TEST_DIRS))

clean: clean-bin clean-deps clean-build-image

clean-bin:
	rm -rf $(BINDIR)

clean-deps:
	rm -f .init

clean-build-image:
	rm -f .kcpBuildImage
	docker rmi -f kcpbuildimage > /dev/null 2>&1 || true

# Building Docker Images for our executables
############################################
images: keepalived-cloud-provider-image

keepalived-cloud-provider-image: $(BINDIR)/keepalived-cloud-provider
	mkdir -p build/keepalived-cloud-provider/tmp
	cp $(BINDIR)/keepalived-cloud-provider build/keepalived-cloud-provider/tmp
	docker build -t $(CONTROLLER_IMAGE) build/keepalived-cloud-provider
	docker tag $(CONTROLLER_IMAGE) $(CONTROLLER_MUTABLE_IMAGE)
	rm -rf build/keepalived-cloud-provider/tmp

# Push our Docker Images to a registry
######################################
push: keepalived-cloud-provider-push

keepalived-cloud-provider-push: keepalived-cloud-provider-image
	docker push $(CONTROLLER_IMAGE)
	docker push $(CONTROLLER_MUTABLE_IMAGE)
