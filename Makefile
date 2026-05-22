.PHONY: setup dev test lint build docker-up docker-down reset-db backend-dev frontend-dev

setup:
	go mod download
	cd frontend && npm install

dev:
	$(MAKE) -j2 backend-dev frontend-dev

backend-dev:
	DARWIN_OPS_MCP_MODE=mock DARWIN_OPS_MCP_SEED_MOCK=true go run ./backend/cmd/server

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
	mkdir -p bin
	go build -o bin/darwin-ops-mcp ./backend/cmd/server
	go build -o bin/darwin-ops-mcp-proxy ./backend/cmd/mcp-proxy
	cd frontend && npm install && npm run build

docker-up:
	docker compose pull backend || true
	docker compose up --build -d
	@echo ""
	@echo "darwin-ops-mcp is starting. Open the frontend at: http://localhost:5173"
	@echo "Backend health check:              http://localhost:8080/healthz"

# Rebuild the backend container locally without compiling Go on this host.
# Dockerfile.backend downloads the CI-built binary from the rolling GitHub release.
docker-up-local-backend:
	docker compose build --no-cache backend
	docker compose up -d backend frontend
	@echo ""
	@echo "Backend rebuilt from GitHub release binary and restarted."
	@echo "Backend health check: http://localhost:8080/healthz"

docker-down:
	docker compose down

reset-db:
	docker compose down -v
	@echo "PostgreSQL volume reset. Run 'make docker-up' to start a fresh stack."
