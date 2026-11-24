.PHONY: dev seed bots logs down reset

dev: .env
	docker compose up -d --wait postgres redis
	docker compose run --rm migrate
	@if [ "$$(docker compose exec -T postgres psql -U hypedomino -d hypedomino -tAc 'select count(*) from config_snapshots')" = "0" ]; then $(MAKE) seed; fi
	docker compose up -d --build --wait server admin-api admin
	docker compose watch --no-up

.env:
	sed "s|^HD_TOKEN_KEYS=.*|HD_TOKEN_KEYS=dev:$$(openssl rand -base64 32)|" .env.example > .env

seed: .env
	docker compose run --rm seed
	docker compose exec -T postgres psql -U hypedomino -d hypedomino -v ON_ERROR_STOP=1 -f /seed/players.sql

bots: .env
	docker compose run --rm seed go run ./cmd/loadbot local $(BOTS)

logs:
	docker compose logs -f server admin-api admin

down:
	docker compose down

reset:
	docker compose down -v
