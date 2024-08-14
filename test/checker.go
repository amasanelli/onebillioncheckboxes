package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/gorilla/websocket"
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

func main() {
	envData = &envSchema{}

	if err := godotenv.Load(ENV_FILE_PATH); err != nil {
		fmt.Println(err)
	}

	if err := env.Parse(envData); err != nil {
		panic(err)
	}

	con, _, err := websocket.DefaultDialer.Dial(envData.WEBSOCKET_URL, nil)
	if err != nil {
		panic(err)
	}
	defer con.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := con.ReadMessage()
			if err != nil {
				panic(err)
			}
			log.Printf("recv: %v", message)
		}
	}()

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			uint32Value := rand.Uint32()%TOTAL_CHECKBOXES + 1

			uint8Slice := make([]uint8, 4)
			binary.LittleEndian.PutUint32(uint8Slice, uint32Value)

			if err := con.WriteMessage(websocket.BinaryMessage, uint8Slice); err != nil {
				panic(err)
			}

			log.Printf("sent: %d", uint32Value)
		case <-sigCh:
			log.Println("bye!")
			return
		}
	}
}
