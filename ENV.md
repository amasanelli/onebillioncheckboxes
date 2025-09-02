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
ALLOWED_ORIGINS="http://localhost:3000"
```

# cloud.docker-compose.env

```
ME_URL="
BUY_ME_A_COFFEE_URL=""
WEBSOCKET_URL="wss://localhost/ws"
ALLOWED_ORIGINS="https://billionchecks.com,https://www.billionchecks.com"
```
