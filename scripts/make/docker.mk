# Docker Compose targets for local development

.PHONY: docker-up docker-down docker-clean docker-logs
.PHONY: mongo-up mongo-down mongo-logs mongo-shell mongo-reset

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-clean:
	docker compose down -v

docker-logs:
	docker compose logs -f

# MongoDB-specific targets
mongo-up:
	docker compose up -d mongodb

mongo-down:
	docker compose down mongodb

mongo-logs:
	docker compose logs -f mongodb

mongo-shell:
	docker compose exec mongodb mongosh -u jobby_app -p jobby_app_pass jobby

mongo-reset:
	docker compose down mongodb -v
	docker compose up -d mongodb
