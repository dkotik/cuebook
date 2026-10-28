PORT ?= 8081

default:
	@go test ./...
serve:
	lsof -tiTCP:$(PORT) -sTCP:LISTEN | while read -r pid; do kill "$$pid"; done
	@reflex -r '\.(go|md|js|css|html)$$' -s -- go run ./cmd/cuebook serve htmx/testdata --port=$(PORT)
agent:
	@reflex -r '\.(go|md|js|css|html)$$' -s -- go run -tags=kronk ./cmd/cuebook serve htmx/testdata --port=$(PORT) --agent
