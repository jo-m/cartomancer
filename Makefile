.PHONY: clean
clean:
	find . -name '*.qtpl.go' -delete
	find . -name '*.gen.go' -delete

.PHONY: db_reset
db_reset:
	rm -f data/db.sqlite

.PHONY: gen
gen: clean
	# Generate.
	go generate ./...

.PHONY: format
format:
	go fmt ./...
	go mod tidy
