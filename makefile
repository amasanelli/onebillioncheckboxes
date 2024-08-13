#!make

start:
	ENV_FILE_PATH="./docker-compose.env" docker-compose -f ./docker-compose.yml up -d
	@echo "All containers started!"