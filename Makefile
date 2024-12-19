
VERSION = $(shell git describe --tags --always)
FLAGS = -ldflags "\
  -X main.VERSION=$(VERSION) \
"

.PHONY: build
build:
	go build $(FLAGS) -o bin/holadoc ./cmd/holadoc

.PHONY: release
release: clean
	GOOS=linux   GOARCH=arm64 go build $(FLAGS) -o bin/holadoc.linux.arm64 ./cmd/holadoc
	GOOS=linux   GOARCH=amd64 go build $(FLAGS) -o bin/holadoc.linux.amd64 ./cmd/holadoc
	GOOS=windows GOARCH=arm64 go build $(FLAGS) -o bin/holadoc.win.arm64.exe ./cmd/holadoc
	GOOS=windows GOARCH=amd64 go build $(FLAGS) -o bin/holadoc.win.amd64.exe ./cmd/holadoc
	GOOS=darwin  GOARCH=arm64 go build $(FLAGS) -o bin/holadoc.mac.arm64 ./cmd/holadoc
	GOOS=darwin  GOARCH=amd64 go build $(FLAGS) -o bin/holadoc.mac.amd64 ./cmd/holadoc
	md5sum bin/* > bin/checksum

.PHONY: run
run:
	go run $(FLAGS) ./cmd/holadoc

.PHONY: serve
serve:
	SERVE=:8080 go run $(FLAGS) ./cmd/holadoc

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: clean
clean:
	rm -f bin/*

.PHONY: deps
deps:
	go get -t -u ./...
	go mod tidy
	go mod vendor
