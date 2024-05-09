VERSION := v0.4.1
PROJECT_ID := olympsis-408521
LOCATION := us-central1-docker.pkg.dev
SERVICE_NAME := server
REPO_NAME := main
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

export BUG_REPORT_COL=bugReports
export POST_REPORT_COL=postReports
export FIELD_REPORT_COL=fieldReports
export EVENT_REPORT_COL=eventReports
export MEMBER_REPORT_COL=memberReports

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
	docker tag $(SERVICE_NAME) $(LOCATION)/$(PROJECT_ID)/$(REPO_NAME)/$(SERVICE_NAME):$(VERSION)
	docker push $(LOCATION)/$(PROJECT_ID)/$(REPO_NAME)/$(SERVICE_NAME):$(VERSION)

local:
	docker build . -t $(SERVICE_NAME)
	docker run -p 80:80 $(SERVICE_NAME):latest

run:
	go run -x main.go

server-up:
	docker images --format '{{.Repository}}:{{.Tag}}' | grep "olympsis-dev-server" | xargs -I {} docker rmi {}
	docker-compose -f tools/dev-compose.yaml up -d

server-down:
	docker-compose -f tools/dev-compose.yaml down

clean: ## Remove previous build
	rm -f $(SERVICE_NAME)