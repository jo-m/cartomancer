.PHONY: clean
clean:
	rm -f internal/pkg/db/{db,models}.go internal/pkg/db/*.sql.go
	find . -name '*.qtpl.go' -delete
	rm -rf tmp
	rm -f cookies.txt

.PHONY: db_reset
db_reset:
	rm -f data/db.sqlite

.PHONY: gen
gen: clean
	# Generate.
	go generate ./...

	# # Format.
	# go fmt ./...
	# go mod tidy
