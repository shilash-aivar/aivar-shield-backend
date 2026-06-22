.PHONY: dev db-up db-down build run test tidy

dev: db-up
	cp -n .env.example .env 2>/dev/null || true
	go run ./cmd/server

db-up:
	docker compose up -d

db-down:
	docker compose down

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...

tidy:
	go mod tidy
