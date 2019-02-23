export GO111MODULE = on

.PHONY: test test-cover

# for test
test:
	go test -race -cover ./...

test-cover:
	go test -race -coverprofile=test.out ./... && go tool cover --html=test.out

release:
	go mod tidy