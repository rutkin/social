#!/bin/bash
set -e

PGPASS_FILE="/var/lib/postgresql/.pgpass"

echo "citus-worker:5432:social:postgres:postgres" > "$PGPASS_FILE"
echo "citus-worker-2:5432:social:postgres:postgres" >> "$PGPASS_FILE"
chown postgres:postgres "$PGPASS_FILE"
chmod 600 "$PGPASS_FILE"