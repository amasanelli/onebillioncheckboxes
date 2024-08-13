package main

import "time"

const (
	PING_INTERVAL      time.Duration = 30 * time.Second
	WRITE_TIMEOUT      time.Duration = 40 * time.Second
	ENV_FILE_PATH      string        = "./.env"
	REDIS_CHANNEL      string        = "ckecks"
	REDIS_KEY          string        = "ckecks"
	TOTAL_CHECKBOXES   uint32        = 1_000_000_000
	RECONNECT_INTERVAL time.Duration = 5 * time.Second
)
