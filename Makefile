.PHONY: build test fuzz vet fmt sync-skills smoke
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
