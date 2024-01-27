VERSION := v0.3.4
SERVICE_NAME := olympsis/server
PKG := "$(SERVICE_NAME)"
PKG_LIST := $( go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $( find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

export PORT=80

export KEYID=JN25FUC9X2
export TEAMID=5A6H49Q85D

export DB_USR=admin
export DB_PASS=vM9pPgfHeZDxgBDv

export DB_NAME=olympsis
export DB_ADDR=database-0.i4q7nvi.mongodb.net
export NOTIF_DB_ADDR=database-2.pdjjqal.mongodb.net
export NOTIF_DB_NAME=notifications
export NOTIF_COL=topics

export AUTH_COL=auth
export USER_COL=users
export CLUB_COL=clubs
export ORG_COL=organizations
export EVENT_COL=events
export FIELD_COL=fields
export POST_COL=posts
export CINVITE_COL=clubInvites
export COMMENTS_COL=comments
export FREQUEST_COL=friendRequests
export CAPPICATIONS_COL=clubApplications
export OAPPICATIONS_COL=organizationApplications

export EVENT_INVITATIONS_COL=eventInvitations
export CLUB_INVITATIONS_COL=clubInvitations
export ORG_INVITATIONS_COL=organizationInvitations

export KEY=SZkp78avQkxGyjRakxb5Ob08zqjguNRA


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