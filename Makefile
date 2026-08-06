.PHONY: test test-cover lint release

test:
	go test -race -cover ./...

test-cover:
	go test -race -coverprofile=test.out ./... && go tool cover --html=test.out

release:
	go mod tidy

lint:
	golangci-lint run
