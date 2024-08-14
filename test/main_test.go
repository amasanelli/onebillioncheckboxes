package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type envSchema struct {
	WEBSOCKET_URL string `env:"WEBSOCKET_URL,required"`
}

const (
	TOTAL_CHECKBOXES uint32 = 1_000_000_000
	ENV_FILE_PATH    string = "./.env"
)

var envData *envSchema

func TestMain(m *testing.M) {
	envData = &envSchema{}

	if err := godotenv.Load(ENV_FILE_PATH); err != nil {
		fmt.Println(err)
	}

	if err := env.Parse(envData); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}
