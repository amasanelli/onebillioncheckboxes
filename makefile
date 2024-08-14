#!make

start:
	./redis/config.sh
	ENV_FILE_PATH="./docker-compose.env" docker-compose -f ./docker-compose.yml up -d redis-1 redis-2 redis-3
	@sleep 5
	echo "yes" | ENV_FILE_PATH="./docker-compose.env" docker-compose -f ./docker-compose.yml exec -T redis-1 /bin/bash -c "redis-cli --cluster create 172.28.0.101:6379 172.28.0.102:6379 172.28.0.103:6379 --cluster-replicas 0"
	@sleep 5
	ENV_FILE_PATH="./docker-compose.env" docker-compose -f ./docker-compose.yml up -d
	@echo "All containers started!"