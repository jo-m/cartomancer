check:
	# Cleanup.
	rm -f data/db.sqlite; mkdir -p data/
	rm -f internal/pkg/db/{db,models}.go internal/pkg/db/*.sql.go
	find . -name '*.qtpl.go' -delete

	# Generate.
	go generate ./...

	# Format.
	go fmt ./...
	go mod tidy
