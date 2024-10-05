VERSION := v0.6.4
PROJECT_ID := olympsis-408521
LOCATION := us-central1-docker.pkg.dev
SERVICE_NAME := server
REPO_NAME := main
PKG := "$(SERVICE_NAME)"
PKG_LIST := $( go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $( find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep build clean test coverage coverhtml lint

all: build

lint: ## Lint the files
	golint -set_exit_status ${PKG_LIST}

test: ## Run unittests
	go test -short ${PKG_LIST}

race: dep ## Run data race detector
	go test -race -short ${PKG_LIST}

msan: dep ## Run memory sanitizer
	go test -msan -short ${PKG_LIST}

dep: ## Get the dependencies
	go get -v -d ./...

build: dep ## Build the binary file
	go build -v $(PKG) 

docker:
	docker build . -f Dockerfile.dev -t $(SERVICE_NAME) --platform linux/amd64 --build-arg VERSION=$(VERSION)
	docker tag $(SERVICE_NAME) $(LOCATION)/$(PROJECT_ID)/$(REPO_NAME)/$(SERVICE_NAME):$(VERSION)
	docker push $(LOCATION)/$(PROJECT_ID)/$(REPO_NAME)/$(SERVICE_NAME):$(VERSION)

local:
	docker build . -f Dockerfile.dev -t $(SERVICE_NAME)
	docker run -p 443:443 $(SERVICE_NAME):latest

run:
	go run -x main.go

server-up:
	docker images --format '{{.Repository}}:{{.Tag}}' | grep "olympsis-dev-server" | xargs -I {} docker rmi {}
	docker-compose -f tools/dev-compose.yaml up -d

server-down:
	docker-compose -f tools/dev-compose.yaml down

clean: ## Remove previous build
	rm -f $(SERVICE_NAME)