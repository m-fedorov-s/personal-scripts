#!/bin/sh
# Ensure the data directory is owned by the appuser
chown -R appuser:appuser /app/data
# Execute the binary as appuser
exec su-exec appuser ./financer.bin
