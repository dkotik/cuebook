default:
	@go test ./...
demo:
	@reflex -r '\.(go|md|js|css|html|cue)$$' -s -- go run ./cmd/htmx-demo -port=8081
agent:
	@reflex -r '\.(go|md|js|css|html|cue)$$' -s -- go run -tags=kronk ./cmd/htmx-demo -port=8081 -agent
