package main

import "time"

const (
	WRITE_TIMEOUT                time.Duration = 10 * time.Second
	READ_TIMEOUT                 time.Duration = 60 * time.Second
	PING_INTERVAL                time.Duration = (READ_TIMEOUT * 9) / 10
	ENV_FILE_PATH                string        = "./.env"
	REDIS_CHANNEL                string        = "checks"
	REDIS_KEY                    string        = "checks"
	TOTAL_CHECKBOXES             uint32        = 1_000_000_000
	RECONNECT_INTERVAL           time.Duration = 5 * time.Second
	REDIS_POOL_SIZE              int           = 100
	REDIS_MAX_ACTIVE_CONNECTIONS int           = REDIS_POOL_SIZE
	BUFFERS_SIZE                 int           = 4096
)
