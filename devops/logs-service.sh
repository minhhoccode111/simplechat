#!/bin/bash

docker compose logs -f "${1:-db}"
