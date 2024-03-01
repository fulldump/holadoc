
.PHONY: build
build:
	go build -o bin/holadoc holadoc

.PHONY: run
run:
	go run holadoc

.PHONY: serve
serve:
	SERVE=:8080 go run holadoc

