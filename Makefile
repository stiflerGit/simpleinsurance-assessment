BUILD_DIR := $(shell pwd)/_out

.PHONY: server client
all: server client

server:
	cd cmd/server && go build -o $(BUILD_DIR)/server

client:
	cd cmd/client && go build -o $(BUILD_DIR)/client

clean:
	#rm -rf $(BUILD_DIR)