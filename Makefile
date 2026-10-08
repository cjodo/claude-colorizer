.PHONY: build install uninstall test manual cross clean

GOBIN_DIR := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)

build:
	go build -o bin/ ./cmd/claude-colorizer

# Installs the binary, then registers the statusline + hooks in ~/.claude/settings.json.
install:
	go install ./cmd/claude-colorizer
	$(GOBIN_DIR)/claude-colorizer install $(ARGS)

uninstall:
	$(GOBIN_DIR)/claude-colorizer uninstall $(ARGS)
	rm -f $(GOBIN_DIR)/claude-colorizer

test:
	go vet ./...
	go test ./...

# Checks what reaches the terminal; see scripts/manual-test.sh. Pass e.g. MANUAL=tmux.
manual:
	scripts/manual-test.sh $(MANUAL)

cross:
	GOOS=linux   GOARCH=amd64 go build -o dist/claude-colorizer-linux-amd64       ./cmd/claude-colorizer
	GOOS=darwin  GOARCH=arm64 go build -o dist/claude-colorizer-darwin-arm64      ./cmd/claude-colorizer
	GOOS=darwin  GOARCH=amd64 go build -o dist/claude-colorizer-darwin-amd64      ./cmd/claude-colorizer
	GOOS=windows GOARCH=amd64 go build -o dist/claude-colorizer-windows-amd64.exe ./cmd/claude-colorizer

clean:
	rm -rf bin dist
