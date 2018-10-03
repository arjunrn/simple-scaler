BIN_DIR := $(GOPATH)/bin
GOMETALINTER := $(BIN_DIR)/gometalinter
PKGS := $(shell go list ./...)
BINARY := scaler
VERSION ?= vlatest
PLATFORMS := windows linux darwin
os = $(word 1, $@)
GO111MODULE=on

.PHONY: test
test: lint
	go test $(PKGS)

$(GOMETALINTER):
	gometalinter --install &> /dev/null

.PHONY: lint
lint: $(GOMETALINTER)
	gometalinter ./...


.PHONY: $(PLATFORMS)
$(PLATFORMS):
	mkdir -p release
	GO111MODULE=on GOOS=$(os) GOARCH=amd64 go build -o release/$(VERSION)-$(os)-amd64/$(BINARY)

.PHONY: release
release: windows linux darwin

clean:
	rm -rf release/


