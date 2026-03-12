COMPOSE := podman-compose

REGISTRY  := quay.io/alexeykazakov
BACKEND   := $(REGISTRY)/cloud-cafe-backend:latest
FRONTEND  := $(REGISTRY)/cloud-cafe-frontend:latest

.PHONY: build up down restart logs ps clean push push-backend push-frontend

build:
	$(COMPOSE) build --no-cache

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down --remove-orphans
	podman rm -f cloud-cafe_frontend_1 cloud-cafe_backend_1 cloud-cafe_postgres_1 2>/dev/null || true

restart: down build up

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

clean: down
	$(COMPOSE) down -v
	podman rmi -f cloud-cafe_backend cloud-cafe_frontend 2>/dev/null || true

push: push-backend push-frontend

push-backend:
	podman tag localhost/cloud-cafe_backend:latest $(BACKEND)
	podman push $(BACKEND)

push-frontend:
	podman tag localhost/cloud-cafe_frontend:latest $(FRONTEND)
	podman push $(FRONTEND)
