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
	gofmt -w .
	go mod tidy

.PHONY: lint
lint:
	gofmt -l .; test -z "$$(gofmt -l .)"
	go vet ./...
	go tool staticcheck -checks=all ./...
	go tool revive -set_exit_status ./...
	go tool gosec ./...
	go tool govulncheck ./...
