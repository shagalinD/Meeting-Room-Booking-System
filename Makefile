all: 

up:
	docker compose up --build

down:
	docker compose down

migrate-up:
	migrate -path ./migrations -database "postgres://postgres:12345678@localhost:5434/postgres?sslmode=disable" up

enter-db:
	docker compose exec database psql -U postgres -d postgres

test:

run:
	docker compose up