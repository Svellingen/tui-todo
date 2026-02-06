.PHONY: build run test lint clean release

build:
	go build -o bin/todo ./cmd/todo

run: build
	./bin/todo

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -rf bin/ dist/

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

release: clean
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build -o dist/todo-$$os-$$arch$$ext ./cmd/todo; \
	done
	@echo "Release binaries in dist/"
