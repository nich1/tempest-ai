.PHONY: help up down logs dev prod build api consumers web migrate sqlc swagger test tidy fmt vet web-dev web-build clean

help:
	@echo "tempest-ai - common targets:"
	@echo "  make dev          - bring up the dev stack (hot-reload, exposed ports)"
	@echo "  make up           - bring up the prod-like stack"
	@echo "  make down         - tear down all containers"
	@echo "  make logs         - tail compose logs"
	@echo "  make scale-consumers N=4 - run N consumer instances"
	@echo "  make api          - run apps/api locally (no docker)"
	@echo "  make consumers    - run apps/consumers locally (no docker)"
	@echo "  make web-dev      - run the Next.js dev server"
	@echo "  make sqlc         - regenerate sqlc code"
	@echo "  make swagger      - regenerate the OpenAPI/Swagger doc"
	@echo "  make migrate      - apply migrations (uses DATABASE_URL or POSTGRES_* env)"
	@echo "  make test         - run go test ./..."
	@echo "  make fmt vet      - go fmt / go vet"
	@echo "  make tidy         - go mod tidy"

dev:
	docker compose -f docker-compose.yml -f docker-compose.local.yml up --build

up:
	docker compose up --build -d

down:
	docker compose -f docker-compose.yml -f docker-compose.local.yml down

logs:
	docker compose logs -f --tail=200

scale-consumers:
	@N=$${N:-4}; docker compose -f docker-compose.yml -f docker-compose.local.yml up --scale consumers=$$N -d

# ----- local dev (no docker) -----
api:
	go run ./apps/api

consumers:
	go run ./apps/consumers

web-dev:
	cd apps/web && npm run dev

web-build:
	cd apps/web && npm run build

# ----- code generation -----
sqlc:
	sqlc generate

swagger:
	swag init -g apps/api/main.go --output docs --parseDependency --parseInternal

migrate:
	go run ./apps/api -migrate-only || true
	@echo "migrate is run at api startup; this target is a placeholder."

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf docs/ tmp/
