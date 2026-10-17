default:
	@go test ./...
demo:
	lsof -tiTCP:8081 -sTCP:LISTEN | while read -r pid; do kill "$$pid"; done
	@reflex -r '\.(go|md|js|css|html)$$' -s -- go run ./cmd/htmx-demo -port=8081
agent:
	@reflex -r '\.(go|md|js|css|html)$$' -s -- go run -tags=kronk ./cmd/htmx-demo -port=8081 -agent
