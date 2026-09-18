# deslop-kitten local task runner. Install via https://just.systems.
# Run `just` for a list of recipes.

# Show available recipes when invoked without arguments.
default:
    @just --list

# Run the full CI pipeline locally. Mirrors .github/workflows/ci.yml.
check: build test-cover lint links

# Compile every package.
build:
    go build ./...

# Run the test suite with the race detector.
test:
    go test -race -timeout 120s ./...

# Like `test` but writes coverage.out. Used by CI.
test-cover:
    go test -race -timeout 120s -coverprofile=coverage.out ./...

# Run golangci-lint v2.
lint:
    golangci-lint run --timeout 5m

# Check markdown links offline. Catches broken local file references.
links:
    lychee --offline '**/*.md'

# Like `links` but also hits external URLs. Slow and rate-limited.
links-online:
    lychee '**/*.md'

# Report gopls' modernize suggestions without changing files.
modernize:
    go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...

# go fmt + go mod tidy.
fmt:
    go fmt ./...
    go mod tidy

# Build bin/deslop-kitten.
binary:
    @mkdir -p bin
    go build -o bin/deslop-kitten ./cmd/deslop-kitten

# Build the lean container image the GitHub Action runs by default.
image:
    docker build -t deslop-kitten:dev .

# Build the judge image, which bundles the llama.cpp server and a 0.8B model.
judge-image:
    docker build -f Dockerfile.judge -t deslop-kitten:judge .

# Score one pull request with the default policy. Needs GITHUB_TOKEN.
score ref:
    go run ./cmd/deslop-kitten score {{ref}}

# Score one pull request inside the judge image. Needs GITHUB_TOKEN.
score-judged ref:
    docker run --rm -e GITHUB_TOKEN deslop-kitten:judge score {{ref}}

# Produce coverage.html for the default browser.
coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "open coverage.html"

# Remove build artifacts.
clean:
    rm -rf bin coverage.out coverage.html
