DOCKER_COMPOSE = docker-compose

start:
	$(DOCKER_COMPOSE) up -d

start-app:
	$(DOCKER_COMPOSE) up --build

stop:
	$(DOCKER_COMPOSE) down

startdb:
	$(DOCKER_COMPOSE) up -d postgres_trafiks redis_trafiks

stopdb:
	$(DOCKER_COMPOSE) stop postgres_trafiks redis_trafiks

build-docker:
	$(DOCKER_COMPOSE) build trafiks

build-ui:
	@echo "Building UI..."
	@cd web/trafiks-ui && npm install && npm run build
	@echo "UI built successfully in routes/dist"

# Build Go binary with embedded UI
# Note: This will embed routes/dist into the binary at compile time
build: build-ui
	@echo "Building Go binary with embedded UI..."
	@go build -o trafiks ./cmd/server/main.go
	@echo "Build complete! Binary includes embedded dashboard."

clean:
	$(DOCKER_COMPOSE) down -v --remove-orphans
	docker volume prune -f

cleandb:
	-docker stop postgres_container redis-x
	-docker rm postgres_container redis-x
	-docker volume rm trafiks_postgres_data trafiks_redis1-data

dev:
	@echo "Starting development environment..."
	@$(DOCKER_COMPOSE) up -d postgres_trafiks redis_trafiks
	@echo "Waiting for databases to be ready..."
	@sleep 2
	@echo "Starting Air for live reload..."
	@air

dev-agent:
	@echo "Starting agent development environment..."
	@$(DOCKER_COMPOSE) up -d postgres_trafiks redis_trafiks
	@echo "Waiting for databases to be ready..."
	@sleep 2
	@echo "Starting Air for agent live reload..."
	@air -c .air.agent.toml

# Docker build and push targets
build-backend:
	@echo "Building UI..."
	@cd web/trafiks-ui && npm install && npm run build
	@echo "Building backend Docker image..."
	@docker build -f Dockerfile.backend -t trafiks-backend:latest .

build-agent:
	@echo "Building agent Docker image..."
	@docker build -f Dockerfile.agent -t trafiks-agent:latest .

build-images: build-backend build-agent
	@echo "All images built successfully!"

push-images:
	@echo "Use ./build-and-push.sh [VERSION] [DOCKERHUB_USERNAME] to build and push images"
	@./build-and-push.sh

.PHONY: start stop startdb stopdb build-docker build clean cleandb dev dev-agent build-ui build-backend build-agent build-images push-images
