#!/bin/bash

set -e

CONFIG_FILE="/etc/redis/redis.conf"
PREFIX="REDIS_"

for ENV_VAR in $(printenv | grep "^${PREFIX}" | awk -F= '{print $1}'); do 
  KEY=$(echo "${ENV_VAR#$PREFIX}" | tr '[:upper:]' '[:lower:]' | tr '_' '-') 

  VALUE="${!ENV_VAR}"
  
  echo $KEY $VALUE

  sed -i "s|^${KEY} .*|${KEY} ${VALUE}|g" "$CONFIG_FILE"
done 

redis-server "$CONFIG_FILE"