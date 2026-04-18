# Docker Compose targets for local development

.PHONY: docker-up docker-up-all docker-down docker-clean docker-logs

docker-up:
	docker compose up -d postgres

docker-up-all:
	docker compose up -d

docker-down:
	docker compose down

docker-clean:
	docker compose down -v

docker-logs:
	docker compose logs -f jobs
