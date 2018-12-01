.PHONY: clean test check build.local build.linux build.osx build.docker build.push

BINARY        ?= dumb-scaler
VERSION       ?= $(shell git describe --tags --always --dirty)
IMAGE         ?= arjunrn/$(BINARY)
TAG           ?= $(VERSION)
SOURCES       = $(shell find . -name '*.go')
DOCKERFILE    ?= Dockerfile
GOPKGS        = $(shell go list ./...)
BUILD_FLAGS   ?= -v
LDFLAGS       ?= -X main.version=$(VERSION) -w -s

default: build.local

clean:
	rm -rf build

test:
	GO111MODULE=on go test -v $(GOPKGS)

check:
	golint $(GOPKGS)
	GO111MODULE=on go vet -v $(GOPKGS)

build.local: build/$(BINARY)
build.linux: build/linux/$(BINARY)
build.osx: build/osx/$(BINARY)

build/$(BINARY): go.mod $(SOURCES)
	GO111MODULE=on CGO_ENABLED=0 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

build/linux/$(BINARY): go.mod $(SOURCES)
	GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/linux/$(BINARY) -ldflags "$(LDFLAGS)" .

build/osx/$(BINARY): go.mod $(SOURCES)
	GO111MODULE=on GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/osx/$(BINARY) -ldflags "$(LDFLAGS)" .

build.docker: build.linux
	docker build --rm -t "$(IMAGE):$(TAG)" -f $(DOCKERFILE) .

build.push: build.docker
	docker push "$(IMAGE):$(TAG)"
