APP_NAME := api-template
CMD_PATH := ./cmd/api
TEMPLATECTL_PATH := ./cmd/templatectl

.PHONY: tidy fmt test run build modules-list modules-doctor

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

modules-list:
	go run $(TEMPLATECTL_PATH) list

modules-doctor:
	go run $(TEMPLATECTL_PATH) doctor
