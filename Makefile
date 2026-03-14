VERSION := v0.8.4
PROJECT_ID := olympsis-485522
LOCATION := us-central1-docker.pkg.dev
SERVICE_NAME := server
PKG := "$(SERVICE_NAME)"
PKG_LIST := $( go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $( find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep build clean test coverage coverhtml lint

all: build

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
	go build -v $(PKG) 

run:
	go run -x main.go

docker-build:
	docker build -f Dockerfile . -t $(SERVICE_NAME)-unsecure

artifact: #Publish image to gcp docker-hub
	docker build . -t $(SERVICE_NAME) --platform linux/amd64 --build-arg VERSION=$(VERSION)
	docker tag $(SERVICE_NAME) $(LOCATION)/$(PROJECT_ID)/$(SERVICE_NAME)/release:$(VERSION)
	docker push $(LOCATION)/$(PROJECT_ID)/$(SERVICE_NAME)/release:$(VERSION)

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

update-service: #Updates the linux service
	make build && \
	if [ $$? -ne 0 ]; then \
		echo "Error: Failed to build new server binary." && \
		exit 1; \
	fi && \
	rm /sbin/olympsis-server && \
	mv olympsis-server /sbin && \
	if [ $$? -ne 0 ]; then \
		echo "Error: Failed to move binary." && \
		exit 1; \
	fi && \
	systemctl restart olympsis-server.service && \
	echo "Update Successful"

clean: ## Remove previous build
	rm -f $(SERVICE_NAME)