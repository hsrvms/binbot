#!/bin/bash
echo "--- Testing PostgreSQL Connection ---"
pg_isready -h postgres -p 5432 -U bot_user

echo -e "\n--- Testing NATS JetStream Connection ---"
curl -s http://localhost:8222/varz | grep "version" || echo "NATS connection failed."
