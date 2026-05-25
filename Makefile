.PHONY: build test lint clean install

build:
	go build -o bin/kfs ./cmd/kfs

install:
	go install ./cmd/kfs

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
