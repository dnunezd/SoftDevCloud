.PHONY: up down logs build test test-integration cover fmt vet mocks

up:            ## Levanta todo el sistema en local
	docker compose up -d --build

down:          ## Detiene y elimina los contenedores
	docker compose down

logs:          ## Sigue los logs de api y worker
	docker compose logs -f api worker

build:         ## Construye las imágenes
	docker compose build

test:          ## Ejecuta las pruebas unitarias
	cd backend && go test -race ./...

cover:         ## Pruebas con reporte de cobertura
	cd backend && go test -covermode=atomic -coverpkg=./internal/... -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

fmt:           ## Formatea el código
	cd backend && gofmt -w .

vet:           ## Análisis estático
	cd backend && go vet ./...

mocks:         ## Regenera los mocks de internal/ports (mockery)
	cd backend && go tool mockery

test-integration: ## Pruebas de repositorios contra PostgreSQL (requiere TEST_DATABASE_URL)
	cd backend && go test -count=1 -tags=integration ./internal/adapters/outbound/postgres/...
