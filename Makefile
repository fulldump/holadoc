
.PHONY: build
build:
	go build -o bin/holadoc main.go

.PHONY: run
run:
	go run main.go

