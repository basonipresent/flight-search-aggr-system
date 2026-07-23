.PHONY: test race lint run cover

## test: run all tests without the race detector
test:
	go test ./...

## race: run all tests with the race detector (required to be clean)
race:
	go test -race ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## run: build and start the server
run:
	go run ./cmd/server

## cover: generate an HTML coverage report and open it
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html
