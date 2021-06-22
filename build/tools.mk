# Install tools used by the build

TOOLSDIR=tools
TOOLSBINDIR=$(TOOLSDIR)/bin
TOOLSSRCDIR=$(TOOLSDIR)/src

GOHOSTOS=$(shell go env GOHOSTOS)
GOHOSTARCH=$(shell go env GOHOSTARCH)

export PATH := $(abspath $(TOOLSBINDIR)):$(PATH)

clobber: clobber-tools

.PHONY: clobber-tools

clobber-tools:
	rm -rf $(TOOLSBINDIR) $(TOOLSDIR)/include $(TOOLSDIR)/*.*

# Protobuf compiler
# =================
#
# Install protoc under $TOOLSDIR. A protoc that is already installed locally
# cannot be trusted since this must be the exact same version as used when
# running CI. If it isn't, the generate-check will fail.
tools/protoc = $(TOOLSBINDIR)/protoc
PROTOC_VERSION=3.13.0
PROTOC_ZIP=protoc-$(PROTOC_VERSION)-$(subst darwin,osx,$(GOHOSTOS))-$(shell uname -m).zip
$(TOOLSDIR)/$(PROTOC_ZIP):
	mkdir -p $(@D)
	curl -sfL https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP) -o $@
%/bin/protoc %/include %/readme.txt: %/$(PROTOC_ZIP)
	cd $* && unzip -q -o -DD $(<F)

# `go get`-able things
# ====================
#
# Install the all under $TOOLSDIR. Versions that are already in $GOBIN
# cannot be trusted since this must be the exact same version as used
# when running CI. If it isn't the generate-check will fail.
#
# Instead of having "VERSION" variables here, the versions are
# controlled by `tools/src/${thing}/go.mod` files.  Having those in
# separate per-tool go.mod files avoids conflicts between tools and
# avoid them poluting our main go.mod file.
tools/protoc-gen-go      = $(TOOLSBINDIR)/protoc-gen-go
tools/protoc-gen-go-grpc = $(TOOLSBINDIR)/protoc-gen-go-grpc
tools/ko                 = $(TOOLSBINDIR)/ko
tools/golangci-lint      = $(TOOLSBINDIR)/golangci-lint
$(TOOLSBINDIR)/%: $(TOOLSSRCDIR)/%/go.mod $(TOOLSSRCDIR)/%/pin.go
	cd $(<D) && $(shell unset GOOS GOARCH; go build -o $(abspath $@)) $$(sed -En 's,^import "(.*)"$$,\1,p' pin.go)

# Protobuf linter
# ===============
#
tools/protolint = $(TOOLSBINDIR)/protolint
PROTOLINT_VERSION=0.26.0
PROTOLINT_TGZ=protolint_$(PROTOLINT_VERSION)_$(shell uname -s)_$(shell uname -m).tar.gz
$(TOOLSDIR)/$(PROTOLINT_TGZ):
	mkdir -p $(@D)
	curl -sfL https://github.com/yoheimuta/protolint/releases/download/v$(PROTOLINT_VERSION)/$(PROTOLINT_TGZ) -o $@
%/bin/protolint %/bin/protoc-gen-protolint: %/$(PROTOLINT_TGZ)
	mkdir -p $(@D)
	tar -C $(@D) -zxmf $< protolint protoc-gen-protolint
