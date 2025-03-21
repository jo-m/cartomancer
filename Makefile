.PHONY: clean
clean:
	find . -name '*_templ.go' -delete
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
	go tool templ fmt .
	go mod tidy

.PHONY: lint
lint:
	gofmt -l .; test -z "$$(gofmt -l .)"
	go vet ./...
	go tool staticcheck -f stylish ./...
	go tool govulncheck ./...
	go tool revive -set_exit_status -formatter stylish ./...
	go tool gosec -exclude G101 ./...

.PHONY: test
test:
	go test -count 1 -v ./...

.PHONY: check
check: gen lint test
