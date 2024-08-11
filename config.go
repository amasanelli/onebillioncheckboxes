package main

import "time"

const (
	pingInterval  time.Duration = 30 * time.Second
	writeTimeout  time.Duration = 40 * time.Second
	ENV_FILE_PATH string        = "./.env"
	CHANNEL       string        = "ckecks"
)
