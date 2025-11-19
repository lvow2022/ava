.PHONY: run worker test tidy

run:
	go run ./cmd/server

worker:
	go run ./cmd/worker

test:
	go test ./...

tidy:
	go mod tidy
