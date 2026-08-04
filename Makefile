all: 

up:
	docker compose up --build

down:
	docker compose down

migrate-up:
	migrate -path ./migrations -database "postgres://postgres:12345678@localhost:5432/postgres" up

test:

run:
	docker compose up