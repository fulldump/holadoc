
.PHONY: build
build:
	go build -o bin/holadoc main.go

.PHONY: run
run:
	go run main.go

.PHONY: serve
serve:
	SERVE=:8080 go run main.go

