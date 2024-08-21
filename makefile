#!make

start-local:
	docker compose -f ./local.docker-compose.yml up -d redis-1 redis-2 redis-3
	@sleep 5
	echo "yes" | docker compose -f ./local.docker-compose.yml exec -T redis-1 /bin/bash -c "redis-cli --cluster create 172.28.0.101:6379 172.28.0.102:6379 172.28.0.103:6379 --cluster-replicas 0"
	@sleep 5
	docker compose -f ./local.docker-compose.yml up -d
	@sleep 5
	@echo "All containers started!"