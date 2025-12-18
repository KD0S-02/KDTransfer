GO_FILES=./...

.PHONY: all build test clean lint

build:
	go build -o bin/kdtransfer ./cmd/client/main.go
	go build -o bin/kdtransfer-server ./cmd/server/main.go	

clean:
	rm -rf bin/*
	go clean