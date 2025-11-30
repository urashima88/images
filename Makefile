run:
	go run ./cmd/app --config=.env --env=local

up:
	docker-compose up -d

build:
	docker-compose up --build

build-app:
	docker-compose up --build images_app

build-db:
	docker-compose up --build postgres_db

MIGRATE_RUN = go run ./cmd/migrator --config=.env --migrations-path=./migrations

migrate:
	$(MIGRATE_RUN)

# forced set version
migrate-force:
	$(MIGRATE_RUN) --force=1

# complete reset of all migrations
migrate-drop:
	$(MIGRATE_RUN) --drop

# show the current version
migrate-version:
	$(MIGRATE_RUN) --version

# roll back one migration
migrate-down:
	$(MIGRATE_RUN) --down

# roll back multiple migrations
migrate-down-steps:
	$(MIGRATE_RUN) --down --steps=2

