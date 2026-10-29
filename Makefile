.PHONY: build test demo

build:
	go build -o else.sh ./cmd/else.sh

test:
	go vet ./... && go test ./...

# Records demo/demo.gif with VHS in Docker, using the stub agent in demo/bin.
demo:
	GOOS=linux GOARCH=$$(go env GOARCH) CGO_ENABLED=0 go build -o demo/bin/else.sh ./cmd/else.sh
	docker run --rm -v "$$PWD":/vhs ghcr.io/charmbracelet/vhs demo/demo.tape
