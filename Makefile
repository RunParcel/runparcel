BINDIR      := $(CURDIR)/bin
BINNAME     := runparcel

GOBIN         = $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN         = $(shell go env GOPATH)/bin
endif
GOX           = $(GOBIN)/gox
GOIMPORTS     = $(GOBIN)/goimports
ARCH          = $(shell go env GOARCH)


# go option
PKG         := ./...
TAGS        :=
TESTS       := .
TESTFLAGS   :=
LDFLAGS     := -w -s
GOFLAGS     :=
CGO_ENABLED ?= 0


.PHONY: all
all: build

.PHONY: build
build:
	@CGO_ENABLED=$(CGO_ENABLED) go build -o $(BINDIR)/$(BINNAME) ./cmd/runparcel

.PHONY: run
run: build
	$(BINDIR)/$(BINNAME) generate --template example/cloudrun/run.yaml.tmpl --values example/values.yaml