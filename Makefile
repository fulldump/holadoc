
VERSION = $(shell git describe --tags --always)
FLAGS = -ldflags "\
  -X main.VERSION=$(VERSION) \
"

.PHONY: build
build:
	go build $(FLAGS) -o bin/holadoc holadoc

.PHONY: release
release: clean
	GOOS=linux   GOARCH=arm64 go build $(FLAGS) -o bin/holadoc.linux.arm64 .
	GOOS=linux   GOARCH=amd64 go build $(FLAGS) -o bin/holadoc.linux.amd64 .
	GOOS=windows GOARCH=arm64 go build $(FLAGS) -o bin/holadoc.win.arm64.exe .
	GOOS=windows GOARCH=amd64 go build $(FLAGS) -o bin/holadoc.win.amd64.exe .
	GOOS=darwin  GOARCH=arm64 go build $(FLAGS) -o bin/holadoc.mac.arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(FLAGS) -o bin/holadoc.mac.amd64 .
	md5sum bin/* > bin/checksum

.PHONY: run
run:
	go run $(FLAGS) holadoc

.PHONY: serve
serve:
	SERVE=:8080 go run $(FLAGS) holadoc

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: clean
clean:
	rm -f bin/*
