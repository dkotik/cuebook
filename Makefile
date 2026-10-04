default:
	@go test ./...
demo:
	@cd htmx && go run -tags=demo -port=8080 .
