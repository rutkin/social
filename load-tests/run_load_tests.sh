#!/bin/bash

# Clean results directory
echo "Cleaning results directory..."
rm -rf ./results/*

# Run load tests and wait for completion
echo "Starting load tests..."
docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit

# Stop containers after tests
echo "Stopping containers..."
docker-compose -f docker-compose.test.yml down 