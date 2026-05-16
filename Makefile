SHELL := bash
.SHELLFLAGS := -o pipefail -c

.PHONY: gen
gen: clean
	# Generate.
	go generate ./...
	mkdir -p static/ && touch static/empty
	go test -run TestOptionsDoc -update-options-doc

.PHONY: dev
dev:
	go tool air

.PHONY: format
format:
	gofmt -w .
	go mod tidy

.PHONY: lint
lint:
	@go mod tidy -diff
	@gofmt -l .; test -z "$$(gofmt -l .)"
	@go vet ./...
	@go tool staticcheck -f stylish ./...
	@go tool revive -set_exit_status -formatter stylish $(shell go list ./... | grep -v 'frontend/')
	@go tool govulncheck ./...
	@go tool gosec -quiet -exclude G101,G304 ./...

.PHONY: test
test:
	go test ./...

.PHONY: test_frontend
test_frontend:
	cd frontend && npm ci && npm test

.PHONY: test_online
test_online:
	go test -v -timeout 5m -count=1 -run=TestOnline --tags=online ./...

.PHONY: bench
bench:
	go test -bench=. -run=Bench ./...

.PHONY: check
check: gen lint
	go build ./...
	@go test ./... | { grep -vE '^(ok|\?)' || true; }
	@cd frontend && npm ci && npm run test -- --silent --reporter=dot

.PHONY: clean
clean:
	find . -name '*.gen.go' -delete

.PHONY: reset_data
reset_data:
	rm -f data/db.sqlite data/geonames.sqlite data/forecast.sqlite
