include .env

migration:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext .sql -dir ./migrations ${name}

up:
	@echo "Running up migrations..."
	migrate -path ./migrations -database ${DB} up

down:
	@echo "Running own migrations..."
	migrate -path ./migrations -database ${DB} down