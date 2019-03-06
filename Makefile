all: help

PKGS := $(shell go list ./... | grep -v /vendor)

PROJECTNAME=$(shell basename "$(PWD)")

## install: Install missing dependencies. Runs `go get` internally.
install:
	@go get $(get)

## test: Runs `go test` on project test files.
.PHONY: test
test:
	go test -v $(PKGS) && echo 'ALL PASS'

## clean: Clean build files. Runs `go clean` internally.
.PHONY: clean
clean:
	@go clean
	@rm -f docker/*.tar
	@rm ipfs/tmp_data

$(GOMETALINTER):
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

## lint: Lints project files, go gets gometalinter if missing. Runs `gometalinter` on project files.
.PHONY: lint
lint: $(GOMETALINTER)
	gometalinter ./... exclude=gosec --vendor

## build: Builds project into an executable binary.
.PHONY: build
build:
	go build -o bin/ipdr cmd/ipdr/ipdr.go

## release: Release a new version. Runs `goreleaser internally.
.PHONY: release
release:
	@rm -rf dist
	goreleaser

.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make command to run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
