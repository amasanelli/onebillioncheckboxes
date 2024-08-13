#! /bin/bash

for port in $(seq 1 3); do
  mkdir -p ./redis/node-${port}
  touch ./redis/node-${port}/redis.conf
  cat << EOF > ./redis/node-${port}/redis.conf
port 6379
bind 0.0.0.0
cluster-enabled yes
cluster-config-file nodes.conf
cluster-node-timeout 5000
cluster-announce-ip 172.28.0.10${port}
cluster-announce-port 6379
cluster-announce-bus-port 16379
appendonly yes
EOF
done