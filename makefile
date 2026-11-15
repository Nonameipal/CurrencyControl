run:
	go run ./cmd/main.go

test:
	go test -v -count=1 ./...

swagger:
	swag init -g internal/delivery/http/routes.go --parseDependency --parseInternal

build:
	swag init -g internal/delivery/http/routes.go --parseDependency --parseInternal
	go build -o app.exe ./cmd/main.go

docker-build:
	docker-compose up -d --build

