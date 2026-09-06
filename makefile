run:
	go run ./cmd/main.go
swagger:
	swag init -g cmd/main.go --output docs --parseDependency --parseInternal

swagg-add:
	git add docs/
swagg-commit:
	git commit -m "Update swagger docs"

swagg-push:
	git push origin HEAD

build:
	swag init -g cmd/main.go --output docs --parseDependency --parseInternal
	go build -o app.exe ./cmd/main.go

docker-build:
	docker-compose up -d --build

DB_URL="postgres://postgres:Noname0212@localhost:5432/currency_control?sslmode=disable"
MIGRATIONS_PATH=./migrations

.PHONY: migrate-up migrate-down migrate-force

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database $(DB_URL) up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database $(DB_URL) down 1

