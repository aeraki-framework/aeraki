# Go parameters
GOCMD?=go
GOBUILD?=$(GOCMD) build
GOTEST?=$(GOCMD) test
GOCLEAN?=$(GOCMD) clean
GOTEST?=$(GOCMD) test
GOGET?=$(GOCMD) get
GOBIN?=$(GOPATH)/bin

# Build parameters
OUT?=./out
DOCKER_TMP?=$(OUT)/docker_temp/
DOCKER_TAG?=aeraki/aeraki:latest
BINARY_NAME?=$(OUT)/aeraki
BINARY_NAME_DARWIN?=$(BINARY_NAME)-darwin
MAIN_PATH_CONSUL_MCP=./cmd/aeraki/main.go

test: style-check
	$(GOTEST) -race ./...	
build: test
	CGO_ENABLED=0 GOOS=linux  $(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH_CONSUL_MCP)
build-mac: test
	CGO_ENABLED=0 GOOS=darwin  $(GOBUILD) -o $(BINARY_NAME_DARWIN) $(MAIN_PATH_CONSUL_MCP)
docker-build: build
	rm -rf $(DOCKER_TMP)
	mkdir $(DOCKER_TMP)
	cp ./docker/Dockerfile $(DOCKER_TMP)
	cp $(BINARY_NAME) $(DOCKER_TMP)
	docker build -t $(DOCKER_TAG) $(DOCKER_TMP)
	rm -rf $(DOCKER_TMP)
docker-push: docker-build
	docker push $(DOCKER_TAG)
clean:
	rm -rf $(OUT)
style-check:
	gofmt -l -d ./
	goimports -l -d ./
lint:
	golint ./...

.PHONY: build docker-build docker-push clean style-check lint
