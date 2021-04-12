BUILD_DIR := $(shell pwd)/_out
PERSISTENCE_DIR := $(shell pwd)/persistence

.PHONY: server client
all: server client

server:
	cd cmd/server && go build -o $(BUILD_DIR)/server

client:
	cd cmd/client && go build -o $(BUILD_DIR)/client

test:
	go test -race ./...

clean:
	rm -rf $(BUILD_DIR) $(PERSISTENCE_DIR)
