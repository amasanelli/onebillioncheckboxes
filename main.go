package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type envSchema struct {
	SERVER_ADDRESS string `env:"SERVER_ADDRESS" validate:"required"`
	REDIS_URI      string `env:"REDIS_URI" validate:"required"`
}

var envData *envSchema
var validate *validator.Validate
var rCli *redis.Client
var pool *connectionsPool
var upg *websocket.Upgrader

func main() {
	envData = &envSchema{}

	validate = validator.New(validator.WithRequiredStructEnabled())

	if err := godotenv.Load(ENV_FILE_PATH); err != nil {
		fmt.Println(err)
	}

	if err := env.Parse(envData); err != nil {
		panic(err)
	}

	if err := validate.Struct(envData); err != nil {
		panic(err)
	}

	upg = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	rOpts, err := redis.ParseURL(envData.REDIS_URI)
	if err != nil {
		panic(fmt.Errorf("error parsing REDIS_URI: %s", err.Error()))
	}

	rCli = redis.NewClient(rOpts)

	if err := rCli.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Errorf("error pinging database: %s", err.Error()))
	}
	defer rCli.Close()

	errCh := make(chan error)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWebsocket)
	mux.HandleFunc("/ping", handlePing)
	mux.HandleFunc("/", handleGet)

	srv := &http.Server{
		Addr:    envData.SERVER_ADDRESS,
		Handler: mux,
	}
	defer srv.Shutdown(context.Background())

	pool = newConnectionsPool()
	defer pool.Close()

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err // error running the http server -> panic
			return
		}
	}()

	fmt.Println("running...")

	select {
	case err := <-errCh:
		log.Panicf("[websockets error]: error running http server: %s\n", err.Error()) // error running the http server -> panic
	case <-sigCh:
		fmt.Println("starting graceful shutdown")
	}
}
