#!/bin/bash

SERVICE="${1:-db}"
docker compose logs -f "$SERVICE"
