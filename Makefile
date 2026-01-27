APP_NAME := joe-ssh
BIN_DIR := bin
HOST_KEY := .ssh/host_ed25519

.PHONY: build run clean build-linux keys fmt

build: | $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) .

run:
	go run .

build-linux: | $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(APP_NAME)-linux .

keys:
	mkdir -p .ssh
	ssh-keygen -t ed25519 -f $(HOST_KEY) -N "" -q
	@echo "Generated host key at $(HOST_KEY)"

fmt:
	go fmt ./...

clean:
	rm -rf $(BIN_DIR)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)
