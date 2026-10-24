run:
	go run ./cmd/server

build:
	go build -o bin/movara ./cmd/server

tidy:
	go mod tidy
