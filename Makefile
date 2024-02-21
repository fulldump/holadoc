
.PHONY: build
build:
	go build -o bin/holadoc main.go

.PHONY: run
run:
	LANGUAGES=en,es,zh go run main.go

.PHONY: serve
serve:
	SERVE=127.0.0.1:8080 go run main.go

