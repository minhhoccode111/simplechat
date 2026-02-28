#!/bin/bash

docker volume rm chat_data
docker compose down -v --remove-orphans
