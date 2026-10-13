.PHONY: run compose-up compose-down

run:
	go run ./cmd/api

compose-up:
	docker compose up --build

compose-down:
	docker compose down
