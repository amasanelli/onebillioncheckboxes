package main

import "time"

const (
	ENV_FILE_PATH                string        = "./.env"
	TOTAL_CHECKBOXES             uint32        = 1_000_000_000
	MAX_CHECKBOXES_PER_REQUEST   uint32        = 100_000
	WRITE_TIMEOUT                time.Duration = 10 * time.Second
	READ_TIMEOUT                 time.Duration = 60 * time.Second
	PING_INTERVAL                time.Duration = (READ_TIMEOUT * 9) / 10
	RECONNECT_INTERVAL           time.Duration = 5 * time.Second
	BUFFERS_SIZE                 int           = 4096
	REDIS_CHANNEL                string        = "checks"
	REDIS_CHECKS_KEY             string        = "checks"
	REDIS_CHECKS_COUNT_KEY       string        = "checks_count"
	REDIS_CHECKS_PER_KEY         uint32        = 100_000_000
	REDIS_POOL_SIZE              int           = 100
	REDIS_MAX_ACTIVE_CONNECTIONS int           = REDIS_POOL_SIZE
)
