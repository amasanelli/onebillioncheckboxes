# .env

```
REDIS_ADDRESSES="172.28.0.101:6379,172.28.0.102:6379,172.28.0.103:6379"
REDIS_ADDRESSES_REMAP="172.28.0.101:6379|:6371,172.28.0.102:6379|:6372,172.28.0.103:6379|:6373"
SERVER_ADDRESS="localhost:3003"
ME_URL=""
BUY_ME_A_COFFEE_URL=""
WEBSOCKET_URL="ws://localhost:3003/ws"
LIMITER_LIMIT=10
LIMITER_BURST=10
```

# cloud.docker-compose.env

```
ME_URL="
BUY_ME_A_COFFEE_URL=""
WEBSOCKET_URL="wss://localhost/ws"
```

# Testing

`k6 -e WEBSOCKET_URL="ws://localhost:3000/ws" run k6.js`

# Self-signed certificate

`openssl req -nodes -keyout onebillioncheckboxes.key -out onebillioncheckboxes.crt -days 365 -new -subj "/CN=onebillioncheckboxes.com" -x509`

# Build and push to Artifact Registry

- Activate Artifact Registry API

## Authenticate docker

`gcloud auth configure-docker australia-southeast2-docker.pkg.dev`

## Build and push app

```bash
docker build --platform linux/amd64 -t "australia-southeast2-docker.pkg.dev/billionchecks/billionchecks/app" .
docker push australia-southeast2-docker.pkg.dev/billionchecks/billionchecks/app
```

## Build and push redis

```bash
docker build --platform linux/amd64 -t "australia-southeast2-docker.pkg.dev/billionchecks/billionchecks/redis" -f ./redis/dockerfile .
docker push australia-southeast2-docker.pkg.dev/billionchecks/billionchecks/redis
```

# Deploying commands

- Activate Compute Engine API
- Create a Debian VM

## Install docker

```bash
sudo apt-get update
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

## Install docker compose

Check the last release at https://github.com/docker/compose/releases using `uname -s` and `uname -m`

```bash
sudo curl -L https://github.com/docker/compose/releases/download/v2.29.2/docker-compose-linux-x86_64 -o /usr/local/bin/docker-compose`
sudo chmod +x /usr/local/bin/docker-compose
```

## Authenticate docker

`sudo gcloud auth configure-docker australia-southeast2-docker.pkg.dev`

# Make some dirs

```bash
sudo mkdir -p /var/www/certbot
sudo mkdir -p /etc/letsencrypt/live/billionchecks.com
```

## Start services

`sudo ENV_FILE_PATH="./cloud.docker-compose.env" docker-compose -f ./cloud.docker-compose.yml up -d redis-1 redis-2 redis-3`

`echo "yes" | sudo ENV_FILE_PATH="./cloud.docker-compose.env" docker-compose -f ./cloud.docker-compose.yml exec -T redis-1 /bin/bash -c "redis-cli --cluster create 172.28.0.101:6379 172.28.0.102:6379 172.28.0.103:6379 --cluster-replicas 0"`

`sudo ENV_FILE_PATH="./cloud.docker-compose.env" docker-compose -f ./cloud.docker-compose.yml up -d`

# DNS

- Reserve a static external IP
- Buy a domain name
- Create a record A and set the value to the IP and the name to "@"
- Create a record CNAME and set the value to the domain and the name to "www"

```bash
sudo apt update
sudo apt install snapd
sudo snap install core
sudo snap refresh core
sudo snap install --classic certbot
sudo ln -s /snap/bin/certbot /usr/bin/certbot
sudo certbot certonly -d billionchecks.com -d www.billionchecks.com --webroot --webroot-path /var/www/certbot --dry-run
sudo certbot certonly -d billionchecks.com -d www.billionchecks.com --webroot --webroot-path /var/www/certbot
```

`sudo systemctl status snap.certbot.renew.service`

`sudo certbot renew --dry-run`
