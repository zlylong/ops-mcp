.PHONY: dev test lint build docker-up docker-down backend-dev frontend-dev

dev:
	$(MAKE) -j2 backend-dev frontend-dev

backend-dev:
	OPS_MCP_MODE=mock go run ./backend/cmd/server

frontend-dev:
	cd frontend && npm install && npm run dev

test:
	go test ./...
	cd frontend && npm install && npm run lint

lint:
	gofmt -w backend
	go vet ./backend/...
	cd frontend && npm install && npm run lint

build:
	go build -o bin/ops-mcp ./backend/cmd/server
	cd frontend && npm install && npm run build

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v
