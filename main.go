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
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type envSchema struct {
	SERVER_ADDRESS        string            `env:"SERVER_ADDRESS,required"`
	REDIS_ADDRESSES       []string          `env:"REDIS_ADDRESSES,required"`
	REDIS_ADDRESSES_REMAP map[string]string `env:"REDIS_ADDRESSES_REMAP" envKeyValSeparator:"|"`
	EMAIL                 string            `env:"EMAIL,required"`
	BUY_ME_A_COFFEE_URL   string            `env:"BUY_ME_A_COFFEE_URL,required"`
	WEBSOCKET_URL         string            `env:"WEBSOCKET_URL,required"`
}

var envData *envSchema
var rCli *redis.ClusterClient
var upgrader *websocket.Upgrader
var indexTemplate *template.Template

func main() {
	var err error

	envData = &envSchema{}

	err = godotenv.Load(ENV_FILE_PATH)
	if err != nil {
		fmt.Println(err)
	}

	err = env.Parse(envData)
	if err != nil {
		panic(fmt.Errorf("error parsing env vars: %s", err.Error()))
	}

	upgrader = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	rCli = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:          envData.REDIS_ADDRESSES,
		PoolSize:       REDIS_POOL_SIZE,
		MaxActiveConns: REDIS_MAX_ACTIVE_CONNECTIONS,
		NewClient: func(opt *redis.Options) *redis.Client {
			if len(envData.REDIS_ADDRESSES_REMAP) > 0 {
				opt.Addr = envData.REDIS_ADDRESSES_REMAP[opt.Addr]
			}
			return redis.NewClient(opt)
		},
	})

	err = rCli.ForEachShard(context.Background(), func(ctx context.Context, shard *redis.Client) error {
		return shard.Ping(ctx).Err()
	})
	if err != nil {
		panic(fmt.Errorf("[redis error]: error pinging database: %s", err.Error()))
	}
	defer rCli.Close()

	indexTemplate, err = template.ParseFiles("./templates/index.html")
	if err != nil {
		panic(fmt.Errorf("[internal server error]: error parsing index template: %s", err.Error()))
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
		log.Panicf("[internal server error]: error running http server: %s\n", err.Error())
	case <-sigCh:
		fmt.Println("starting graceful shutdown...")
	}
}
