default:
	@go test ./...
demo:
	@go run ./cmd/htmx-demo -port=8081
