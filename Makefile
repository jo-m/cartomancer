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
	go test -run TestOptionsDoc -update-options-doc

.PHONY: format
format:
	gofmt -w .
	go mod tidy

.PHONY: lint
lint:
	gofmt -l .; test -z "$$(gofmt -l .)"
	go vet ./...
	go tool staticcheck -f stylish ./...
	go tool revive -set_exit_status -formatter stylish $(shell go list ./... | grep -v 'frontend/')
	go tool govulncheck ./...
	# TODO: Eventually activate this.
	# go tool gosec -exclude G101,G304 ./...

.PHONY: test
test:
	go test ./...

.PHONY: test_online
test_online:
	go test -v -timeout 5m -count=1 -run=TestOnline --tags=online ./...

.PHONY: bench
bench:
	go test -bench=. -run=Bench ./...

.PHONY: frontend
frontend:
	cd frontend && npm ci && npm run build

.PHONY: check
check: gen format lint
	go build ./...
	go test ./...
