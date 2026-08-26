run:
	go run ./cmd/main.go
swagger:
	swag init -g cmd/main.go

swagg-add:
	git add docs/
swagg-commit:
	git commit -m "Update swagger docs"

swagg-push:
	git push origin HEAD