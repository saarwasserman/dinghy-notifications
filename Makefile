include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	@go run ./cmd/server

.PHONY: run/consumer
run/consumer:
	@go run ./cmd/email/consumer

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy and vendor dependencies and format, vet and test all code
.PHONY: audit
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api

.PHONY: build/consumer
build/consumer:
	@echo 'Building cmd/email/consumer...'
	go build -ldflags='-s' -o=./bin/consumer ./cmd/email/consumer
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/consumer ./cmd/email/consumer

.PHONY: build/docker/api
build/docker/api:
	@echo 'Building build/docker/api...'
	docker build -f build/Dockerfile_api  --tag dinghy/notifications-api:0.1.0 .

.PHONY: build/docker/consumer
build/docker/consumer:
	@echo 'Building build/docker/consumer...'
	docker build -f build/Dockerfile_consumer  --tag dinghy/notifications-consumer:0.1.0 .