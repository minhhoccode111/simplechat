#!/bin/bash

docker network rm chat_network
docker volume rm chat_data
docker compose down -v --remove-orphans
