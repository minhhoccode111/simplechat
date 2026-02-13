#!/bin/bash

SERVICE="${1:-db}"
docker compose exec "$SERVICE" sh
