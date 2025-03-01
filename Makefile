check:
	rm -f data/db.sqlite
	go tool goose up
	go tool sqlc generate
	go tool sqlc vet
	find . -name '*.qtpl.go' -delete
	go generate ./...
	go fmt ./...
	go mod tidy
