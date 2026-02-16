.PHONY: build linux run clean deps

BINARY=boxy

build:
	go build -o $(BINARY) ./cmd/boxy

linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux ./cmd/boxy

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

deps:
	go mod tidy
