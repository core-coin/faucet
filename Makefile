export DOCKER_BUILDKIT = 1
export COMPOSE_DOCKER_CLI_BUILD = 1
include .makeenv

generate:
	@docker build \
		--build-arg=GIT_ENERGY_USERNAME \
		--build-arg=GIT_ENERGY_PASSWORD \
		--target generate --output . .

build:
	@export docker build \
		--build-arg=GIT_ENERGY_USERNAME \
		--build-arg=GIT_ENERGY_PASSWORD \
		--target build --output . .

build-image:
	@docker build \
		--build-arg=GIT_ENERGY_USERNAME \
		--build-arg=GIT_ENERGY_PASSWORD \
		--tag faucet:latest .

server-start:
	@docker-compose up --detach

server-stop:
	@docker-compose down

server-reload:
	@docker-compose up --build --detach faucet

server-logs:
	@docker-compose logs --follow; true
