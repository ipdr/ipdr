all: help

PKGS := $(shell go list ./... | grep -v /vendor)

PROJECTNAME=$(shell basename "$(PWD)")

## install: Install missing dependencies. Runs `go get` internally.
install:
	@go get $(get)

DOCKER_VERSION=18.03.1-ce

.PHONY: test/install-deps
test/install-deps:
	set -x
	# install Docker
	curl -L -o /tmp/docker-$(DOCKER_VERSION).tgz https://download.docker.com/linux/static/stable/x86_64/docker-$$DOCKER_VERSION.tgz
	tar -xz -C /tmp -f /tmp/docker-$(DOCKER_VERSION).tgz
	sudo mv /tmp/docker/* /usr/bin
	# install IPFS
	wget https://dist.ipfs.io/go-ipfs/v0.4.14/go-ipfs_v0.4.14_linux-amd64.tar.gz -O /tmp/go-ipfs.tar.gz
	cd /tmp
	tar xvfz go-ipfs.tar.gz
	sudo cp go-ipfs/ipfs /usr/bin/
	ipfs version
	# run IPFS daemon
	ipfs init
	ipfs config Addresses.API /ip4/0.0.0.0/tcp/5001
	ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/9001
	ipfs daemon &

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
	go build -o bin/ipdr cmd/ipdr/main.go

## release: Release a new version. Runs `goreleaser internally.
.PHONY: release
release:
	@rm -rf dist
	goreleaser cmd/ipdr/main.go

.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make command to run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
