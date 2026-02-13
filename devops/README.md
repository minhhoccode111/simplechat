# DevOps

## Makefile

Run `make help` to see available commands.

**Start/stop:**

- `make up-all` / `make down-all` - All services
- `make be-up` / `make be-down` - Backend + database
- `make db-up` / `make db-down` - Database only

## Scripts

- `./init-volume-network.sh` - initialize persistent volume and network
- `./rm-volume-network.sh` - remove persistent volume and network
- `./logs-service.sh <service_name>` - check logs a service
- `./exec-shell-service.sh <service_name>` - execute shell in service
- `./nuke-all.sh` - remove everything
