PLATFORMS := "linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"

_help:
    @just --list -f {{ justfile() }} --list-heading '' --unsorted

# List available recipes
default:
    @just --list

# Build binary into bin/
build:
    go build -o bin/todo ./cmd/todo

# Build and run
run: build
    ./bin/todo

# Run all tests
test:
    go test ./... -v

# Run go vet
lint:
    go vet ./...

# Remove build artifacts
clean:
    rm -rf bin/ dist/

# Cross-compile release binaries for all platforms
release: clean
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p dist
    for platform in {{ PLATFORMS }}; do
        os="${platform%/*}"
        arch="${platform#*/}"
        ext=""
        if [ "$os" = "windows" ]; then ext=".exe"; fi
        echo "Building $os/$arch..."
        GOOS="$os" GOARCH="$arch" go build -o "dist/todo-$os-$arch$ext" ./cmd/todo
    done
    echo "Release binaries in dist/"
