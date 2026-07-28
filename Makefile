BINARY := cowyo2
NODE_MODULES_LOCK := node_modules/.package-lock.json
AIR_VERSION := v1.65.1
SQLC_VERSION := v1.31.1

.PHONY: build frontend generate migrate serve test

build: frontend
	go build -trimpath -ldflags="-s -w" -o $(BINARY) .

frontend: $(NODE_MODULES_LOCK)
	npm run build

$(NODE_MODULES_LOCK): package.json package-lock.json
	npm install

generate:
	CGO_ENABLED=0 go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

migrate:
	CGO_ENABLED=0 go run ./cmd/migrate

serve:
	go run github.com/air-verse/air@$(AIR_VERSION) -c .air.toml

test: frontend
	npm test
	go test ./...
