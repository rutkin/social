#!/bin/bash

# Get the absolute path of the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Check if CSV file path is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <path_to_csv_file>"
    echo "Example: $0 /path/to/your/users.csv"
    exit 1
fi

CSV_FILE="$1"

# Check if PostgreSQL container is running
if ! docker ps | grep -q "postgres"; then
    echo "Starting PostgreSQL container..."
    docker-compose up -d db
    sleep 5
fi

# Use the correct container name
CONTAINER_NAME="social-db-1"

echo "Using container: $CONTAINER_NAME"

# Check if CSV file exists
if [ ! -f "$CSV_FILE" ]; then
    echo "Error: CSV file not found at $CSV_FILE"
    exit 1
fi

# Check if SQL script exists
if [ ! -f "$SCRIPT_DIR/import_users.sql" ]; then
    echo "Error: SQL script not found at $SCRIPT_DIR/import_users.sql"
    exit 1
fi

# Convert CSV file to UTF-8 without BOM
echo "Converting CSV file to UTF-8..."
TEMP_CSV="/tmp/people_utf8.v2.csv"
# First try to detect the encoding
ENCODING=$(file -I "$CSV_FILE" | cut -d'=' -f2)
if [ -z "$ENCODING" ]; then
    ENCODING="WINDOWS-1251"  # fallback to common Russian encoding
fi
echo "Detected encoding: $ENCODING"

# Convert to UTF-8 without BOM
iconv -f "$ENCODING" -t UTF-8 "$CSV_FILE" | tr -d '\r' > "$TEMP_CSV"

# Copy converted CSV file to container
echo "Copying CSV file to container..."
docker cp "$TEMP_CSV" "$CONTAINER_NAME:/tmp/people.v2.csv"

# Copy SQL script to container
echo "Copying SQL script to container..."
docker cp "$SCRIPT_DIR/import_users.sql" "$CONTAINER_NAME:/tmp/import_users.sql"

# Run import script
echo "Running import script..."
docker exec -i "$CONTAINER_NAME" psql -U postgres -d social -f /tmp/import_users.sql

# Clean up temporary file
rm "$TEMP_CSV"

# Verify import
echo "Verifying import..."
docker exec -i "$CONTAINER_NAME" psql -U postgres -d social -c "SELECT COUNT(*) FROM users;"

echo "Import completed!" 