BINARY  := rulebound
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/armstrongl/rulebound/cmd.Version=$(VERSION)"

.PHONY: build test clean install

## build: Compile the rulebound binary into ./rulebound
build:
	go build $(LDFLAGS) -o $(BINARY) .

## test: Run all tests
test:
	go test ./...

## clean: Remove the compiled binary
clean:
	rm -f $(BINARY)

## install: Install the binary to $GOPATH/bin (or $GOBIN)
install:
	go install $(LDFLAGS) .
