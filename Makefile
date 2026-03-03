.PHONY: clean
clean:
	find . -name '*.gen.go' -delete

.PHONY: reset_data
reset_data:
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
	go tool staticcheck -f stylish ./...
	# TODO: Activate again.
	# go tool revive -set_exit_status -formatter stylish ./...
	# go tool gosec -exclude G101 ./...
	# go tool govulncheck ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: bench
bench:
	go test -v -bench=. -run=Bench ./...

.PHONY: check
check: gen format lint test
