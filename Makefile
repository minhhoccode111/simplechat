LOCAL_STACK = docker compose

# HELP =========================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

db-up: ### Start db service
	$(LOCAL_STACK) up -d db
.PHONY: db-up

db-down: ### Stop db service
	$(LOCAL_STACK) down db
.PHONY: db-down

be-up: ### Start db + be services
	$(LOCAL_STACK) up -d db be
.PHONY: be-up

be-down: ### Stop db + be services
	$(LOCAL_STACK) down db be
.PHONY: be-down

up-all: ### Start all services (db + be + fe + nginx)
	$(LOCAL_STACK) up --build -d && $(LOCAL_STACK) logs -f
.PHONY: up-all

down-all: ### Stop all services (db + be + fe + nginx)
	$(LOCAL_STACK) down --remove-orphans
.PHONY: down-all
