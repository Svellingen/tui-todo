.PHONY: build run test clean

build:
	go build -o bin/todo ./cmd/todo

run: build
	./bin/todo

test:
	go test ./... -v

clean:
	rm -rf bin/
