APP_NAME := api-template
CMD_PATH := ./cmd/api

.PHONY: tidy fmt test run build

tidy:
	go mod tidy

fmt:
	go fmt ./...

test:
	go test ./...

run:
	go run $(CMD_PATH)

build:
	go build -o ./bin/$(APP_NAME) $(CMD_PATH)
