.PHONY: build run clean deps

BINARY=boxy

build:
	go build -o $(BINARY) ./cmd/boxy

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

deps:
	go mod tidy
