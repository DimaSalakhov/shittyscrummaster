ARTIFACT = shittyscrummaster
# DOCKER_TAG = $(shell git rev-parse --short HEAD)
DOCKER_ARTIFACT = $(ARTIFACT)-linux-amd64

SRCS = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

build : $(SRCS)
	@echo ">> Building"
	go build -o $(ARTIFACT) .

run : build
	./$(ARTIFACT)

.PHONY: docker-build
docker-build :
	@echo ">> Building"
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.gitRev=$(DOCKER_TAG)" -o $(DOCKER_ARTIFACT) -a .
