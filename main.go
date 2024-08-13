package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/template"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type envSchema struct {
	SERVER_ADDRESS      string `env:"SERVER_ADDRESS" validate:"required"`
	REDIS_URI           string `env:"REDIS_URI" validate:"required"`
	EMAIL               string `env:"EMAIL" validate:"required"`
	BUY_ME_A_COFFEE_URL string `env:"BUY_ME_A_COFFEE_URL" validate:"required"`
	WEBSOCKET_URL       string `env:"WEBSOCKET_URL" validate:"required"`
}

var envData *envSchema
var validate *validator.Validate
var rCli *redis.Client
var upgrader *websocket.Upgrader
var indexTemplate *template.Template

func main() {
	var err error

	envData = &envSchema{}

	validate = validator.New(validator.WithRequiredStructEnabled())

	err = godotenv.Load(ENV_FILE_PATH)
	if err != nil {
		fmt.Println(err)
	}

	err = env.Parse(envData)
	if err != nil {
		panic(fmt.Errorf("error parsing env vars: %s", err.Error()))
	}

	err = validate.Struct(envData)
	if err != nil {
		panic(fmt.Errorf("invalid env vars: %s", err.Error()))
	}

	upgrader = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	rOpts, err := redis.ParseURL(envData.REDIS_URI)
	if err != nil {
		panic(fmt.Errorf("error parsing REDIS_URI: %s", err.Error()))
	}

	rCli = redis.NewClient(rOpts)

	err = rCli.Ping(context.Background()).Err()
	if err != nil {
		panic(fmt.Errorf("error pinging database: %s", err.Error()))
	}
	defer rCli.Close()

	indexTemplate, err = template.ParseFiles("./templates/index.html")
	if err != nil {
		panic(fmt.Errorf("error parsing index template: %s", err.Error()))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWebsocket)
	mux.HandleFunc("/ping", handlePing)
	mux.HandleFunc("/", handleGet)

	srv := &http.Server{
		Addr:    envData.SERVER_ADDRESS,
		Handler: mux,
	}
	defer srv.Shutdown(context.Background())

	errCh := make(chan error)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
			return
		}
	}()

	fmt.Println("running...")

	select {
	case err = <-errCh:
		log.Panicf("error running http server: %s\n", err.Error())
	case <-sigCh:
		fmt.Println("starting graceful shutdown...")
	}
}
