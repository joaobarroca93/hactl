BIN     := hactl
PREFIX  := /usr/local/bin

.PHONY: build install test test-v cover clean

build:
	go build -o $(BIN) .

install: build
	sudo mv $(BIN) $(PREFIX)/$(BIN)

test:
	go test ./...

test-v:
	go test -v ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -f $(BIN) coverage.out
