default:
	@go test ./...
demo:
	@go run -tags=demo ./cmd/htmx-demo -port=8081
