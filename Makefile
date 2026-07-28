BINARY := cowyo
WEB_DIR := web
NODE_MODULES_LOCK := $(WEB_DIR)/node_modules/.package-lock.json
AIR_VERSION := v1.65.1
SQLC_VERSION := v1.31.1

.PHONY: build frontend generate migrate serve test

build: frontend
	go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/cowyo

frontend: $(NODE_MODULES_LOCK)
	npm --prefix $(WEB_DIR) run build

$(NODE_MODULES_LOCK): $(WEB_DIR)/package.json $(WEB_DIR)/package-lock.json
	npm --prefix $(WEB_DIR) ci

generate:
	CGO_ENABLED=0 go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

migrate:
	CGO_ENABLED=0 go run ./cmd/migrate

serve:
	go run github.com/air-verse/air@$(AIR_VERSION) -c .air.toml

test: frontend
	npm --prefix $(WEB_DIR) test
	go test ./...
