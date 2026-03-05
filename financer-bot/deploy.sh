#!/bin/bash
set -e

# Ensure BOT_TOKEN is set
if [ -z "$BOT_TOKEN" ]; then
  echo "Error: BOT_TOKEN environment variable is not set."
  exit 1
fi

# Build and deploy
docker compose build
docker compose up -d --force-recreate
echo "Done!"
