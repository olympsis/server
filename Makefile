# VERSION is derived from git, not hand-edited — the tag is the single source
# of truth (see package version and .github/workflows/release.yml). An untagged
# or dirty tree builds as e.g. "v0.9.4-8-g6351f14-dirty", which is a useful
# signal in itself: it means "not a real release".
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Linker flags that weld the build identity into the binary. Deliberately not
# env vars: an env var can change without a rebuild, letting the server report
# a version it is not actually running.
LDFLAGS := -s -w \
	-X olympsis-server/version.Version=$(VERSION) \
	-X olympsis-server/version.Commit=$(COMMIT) \
	-X olympsis-server/version.BuildTime=$(BUILD_TIME)

PROJECT_ID := olympsis-485522
AR_LOCATION := northamerica-northeast1
LOCATION := northamerica-northeast1-docker.pkg.dev
# Trailing image name is required by AR: PROJECT/REPOSITORY/IMAGE.
DOCKER_IMAGE := $(LOCATION)/$(PROJECT_ID)/docker-images/server
BIN_REPO := go-binaries
BIN_NAME := olympsis-server
SERVICE_NAME := server
PKG := "$(SERVICE_NAME)"
PKG_LIST := $( go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $( find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep build clean test coverage coverhtml lint proto release artifact deploy-mac-mini

all: build

# Regenerate Go from the protos: eventteam.proto (hosted here, called by
# invite-service) -> grpcapi/eventteampb, and invite.proto (a copy of
# invite-service's contract, called from here at check-in) -> grpcapi/invitepb.
# Dev-only: the generated *.pb.go files are committed so the hosting box just
# needs `go build`. Requires protoc + protoc-gen-go + protoc-gen-go-grpc on PATH
# (see invite-service/Makefile for the one-time install commands).
proto:
	PATH="$$PATH:$$(go env GOPATH)/bin" protoc \
		--go_out=. --go_opt=module=olympsis-server \
		--go-grpc_out=. --go-grpc_opt=module=olympsis-server \
		grpcapi/eventteam.proto grpcapi/invite.proto

lint: ## Lint the files
	golint -set_exit_status ${PKG_LIST}

test: ## Run unit tests
	go test -short ${PKG_LIST}

race: dep ## Run data race detector
	go test -race -short ${PKG_LIST}

msan: dep ## Run memory sanitizer
	go test -msan -short ${PKG_LIST}

dep: ## Get the dependencies
	go get -v -d ./...

build: dep ## Build the binary file
	go build -v -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_NAME) .

run:
	go run -x main.go

docker-build:
	docker build -f Dockerfile . -t $(SERVICE_NAME)-unsecure

artifact: #Manual image publish. Prefer `make release` — CD does this reproducibly.
	docker build . --platform linux/amd64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):$(VERSION)

release: #Cut a release: tag and push. GitHub Actions builds and publishes.
	@test -n "$(V)" || { echo "usage: make release V=v0.9.5"; exit 1; }
	@git diff --quiet || { echo "working tree is dirty - commit first"; exit 1; }
	git tag -a $(V) -m "release: $(V)"
	git push origin $(V)
	@echo "Tagged $(V). Watch: https://github.com/olympsis/server/actions"

# Pull a published binary onto the Mac mini and restart pm2.
#
# Prerequisites on the mini:
#   - gcloud installed and authenticated with artifactregistry.reader
#   - SSH there is PASSWORD auth (no key) and sudo prompts again — both are
#     interactive, hence `ssh -tt` to allocate a pty.
#   - pm2's daemon runs as ROOT with PM2_HOME=/Users/joel/.pm2, and sudo's
#     secure PATH excludes Homebrew, so pm2 needs an explicit PATH and HOME or
#     it fails with "command not found".
#
# `--destination` is a DIRECTORY, so the file is staged there and then renamed
# into place. The rename is atomic and works
# while the old binary is executing (the live process keeps the old inode),
# which avoids the ETXTBSY you would hit overwriting it in place.
deploy-mac-mini:
	@test -n "$(V)" || { echo "usage: make deploy-mac-mini V=v0.9.5"; exit 1; }
	ssh -tt joel@joels-mac-mini '\
		set -e; \
		cd /Users/joel/Documents/olympsis-platform/builds && \
		mkdir -p .staging && \
		gcloud artifacts generic download --project=$(PROJECT_ID) \
			--location=$(AR_LOCATION) --repository=$(BIN_REPO) \
			--package=$(BIN_NAME) --version=$(V) \
			--name=$(BIN_NAME) --destination=.staging && \
		mv .staging/$(BIN_NAME) $(BIN_NAME) && \
		chmod +x $(BIN_NAME) && \
		sudo env PATH=/opt/homebrew/bin:$$PATH HOME=/Users/joel pm2 restart server && \
		sudo env PATH=/opt/homebrew/bin:$$PATH HOME=/Users/joel pm2 logs server --nostream --lines 15'

server: #Secure server with local CA certificates
	docker images --format '{{.Repository}}:{{.Tag}}' | grep "$(SERVICE_NAME)" | xargs -I {} docker rmi {}
	docker build -f Dockerfile --secret id=crt,src=./tools/localhost.crt --secret id=key,src=./tools/localhost.key . -t $(SERVICE_NAME)
	docker run \
		--env-file .env.prod \
		-p 443:443 $(SERVICE_NAME):latest

unsecure-server: #Un-secure server with http
	docker images --format '{{.Repository}}:{{.Tag}}' | grep "$(SERVICE_NAME)-unsecure" | xargs -I {} docker rmi -f {}
	docker build -f Dockerfile . -t $(SERVICE_NAME)-unsecure
	docker run \
		-v $(PWD)/files/AuthKey_5MP3VW78BZ.p8:/app/AuthKey_5MP3VW78BZ.p8:ro \
		-v $(PWD)/files/firebase-credentials.json:/app/firebase-credentials.json:ro \
		--env-file .env.dev \
		-p 80:80 $(SERVICE_NAME)-unsecure:latest

dev-up: #Runs the docker-compose stack to set up local environment
	docker images --format '{{.Repository}}:{{.Tag}}' | grep "olympsis-dev-server" | xargs -I {} docker rmi {}
	docker-compose -f compose.dev.yaml up -d

dev-down: #Takes down the docker-compose stack
	docker-compose -f compose.dev.yaml down

prod-up:
	docker-compose -f compose.yaml up -d

prod-down:
	docker-compose -f compose.yaml down

mac-mini: #DEPRECATED. Superseded by `make release` + `make deploy-mac-mini V=...`.
	@echo "This target committed a 42MB binary to git-lfs on every release."
	@echo "Use instead:"
	@echo "  make release V=v0.9.5           # tag + push; CD builds and publishes"
	@echo "  make deploy-mac-mini V=v0.9.5   # pull that build onto the mini"
	@exit 1

clean: ## Remove previous build
	rm -f $(BIN_NAME)