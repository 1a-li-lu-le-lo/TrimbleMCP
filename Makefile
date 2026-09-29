.PHONY: build test fuzz vet fmt sync-skills smoke catalog
build:
	go build -o bin/ ./cmd/...
test:
	go test -race -count=1 ./...
vet:
	go vet ./...
fmt:
	gofmt -w .
fuzz:
	go test ./internal/domain -run=XXX -fuzz=FuzzIDNeverAllowsPathOrQueryBreakout -fuzztime=30s
	go test ./internal/mcp -run=XXX -fuzz=FuzzStdioNeverPanicsAndEmitsOnlyJSON -fuzztime=30s
sync-skills:
	./scripts/sync-skills.sh
smoke: build
	./scripts/smoke.sh
catalog:
	./scripts/fetch-trimble-specs.sh /tmp/trimble-specs
	go run ./cmd/trimble-catalog -specs /tmp/trimble-specs -index /tmp/trimble-specs/index.json -regions /tmp/trimble-specs/regions.json -retrieved $$(date -u +%Y-%m-%d)
