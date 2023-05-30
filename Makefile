VERSION 	 := v0.1.5
SERVICE_NAME := olympsis/server
PKG := "$(SERVICE_NAME)"
PKG_LIST := $( go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $( find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

export PORT=80
export DB_ADDR=192.168.1.205
export DB_USR=service
export DB_PASS=qN1PHHgo6L942AvpTgGQ
export DB_NAME=olympsis
export STORAGE_ADDR=192.168.1.205:9000
export STORAGE_ACCESS_KEY=p4eHM3a4v3wGB2ro
export STORAGE_SECRET_KEY=WbPjehYtD3wO4V4PNlYwJwWiPRy6qqqN
export AUTH_COL=auth
export USER_COL=users
export CLUB_COL=clubs
export EVENT_COL=events
export FIELD_COL=fields
export POST_COL=posts
export CINVITE_COL=clubInvites
export COMMENTS_COL=comments
export FREQUEST_COL=friendRequests
export CAPPICATIONS_COL=clubApplications
export KEY=SZkp78avQkxGyjRakxb5Ob08zqjguNRA

export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=20031998
export TOPIC_DB_NAME=olympsis_notif

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
	docker build . -t $(SERVICE_NAME) --platform linux/amd64 --build-arg VERSION=$(VERSION)
	docker tag $(SERVICE_NAME):latest $(SERVICE_NAME):$(VERSION)
	docker push $(SERVICE_NAME):$(VERSION)

local:
	docker build . -t $(SERVICE_NAME)
	docker run -p 8080:8080 $(SERVICE_NAME):latest

run:
	go run -x main.go

mongo:
	docker run -p 27017:27017 -d -e MONGO_INITDB_ROOT_USERNAME=service -e MONGO_INITDB_ROOT_PASSWORD=qN1PHHgo6L942AvpTgGQ mongo:latest

clean: ## Remove previous build
	rm -f $(SERVICE_NAME)