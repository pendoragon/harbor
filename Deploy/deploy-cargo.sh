#!/bin/bash

# exit immediately if a command exits with a non-zero status.
set -e

echo "This shell will launch Harbor project on local with cargo config"
echo "Usage: ./depoly-cargo.sh"

echo "config hostname..."
sed 's/hostname = reg.mydomain.com/hostname = localhost/g' -i ./harbor.cfg

echo "prepare config..."
./prepare

echo "nginx listen on 8002..."
sed 's/listen 80;/listen 8002;/g' -i ./config/nginx/nginx.conf

echo "build gobase for harbor/ui and harbor/job..."
docker build -f ../Dockerfile.gobase -t gobase:latest ../

echo "docker-compose up cargo..."
docker-compose -f docker-compose-cargo.yml up -d
