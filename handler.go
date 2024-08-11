package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type handler struct {
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	upg := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	con, err := upg.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[websockets error]: upgrade error: %s\n", err.Error())
		return
	}
	// The default close handler echoes the close code to the client (follows the closing handshake)
	// So if the client sends a close message (we read messages using con.ReadMessage()), the closing message is going to be sent back and then a CloseError is thrown
	// If a different kind of error is thrown, the connection is closed without sending or waiting for a close message
	defer con.Close()

	pool.Add(con)
	defer pool.Remove(con)

	mutex := sync.Mutex{}

	con.SetPingHandler(func(msg string) error {
		mutex.Lock()
		err := con.WriteMessage(websocket.PongMessage, []byte(msg))
		mutex.Unlock()
		if err != nil {
			log.Printf("[websockets error]: error sending pong message to client: %s\n", err.Error())
			return err
		}
		return nil
	})

	con.SetWriteDeadline(time.Now().Add(writeTimeout))

	con.SetPongHandler(func(string) error {
		con.SetWriteDeadline(time.Now().Add(writeTimeout))
		return nil
	})

	pubsub := rCli.Subscribe(ctx, CHANNEL)
	defer pubsub.Close()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	go func() {
		for {
			_, data, err := con.ReadMessage()
			if err != nil {
				log.Printf("[websockets error]: error reading message from client: %s\n", err.Error())
				cancel()
				return
			}

			dataLen := len(data)

			if dataLen != 4 && dataLen != 8 {
				continue
			}

			uint32SliceLen := dataLen / 4
			uint32Slice := make([]uint32, uint32SliceLen)

			for i := 0; i < uint32SliceLen; i++ {
				start := i * 4
				uint32Slice[i] = binary.LittleEndian.Uint32(data[start:])
			}

			if uint32SliceLen == 1 {
				if err := rCli.Publish(ctx, CHANNEL, data).Err(); err != nil {
					fmt.Println(err)
				}

				value := strconv.FormatUint(uint64(uint32Slice[0]), 10)
				if err := rCli.ZAdd(ctx, CHANNEL, redis.Z{Score: float64(uint32Slice[0]), Member: value}).Err(); err != nil {
					fmt.Println(err)
				}
			} else {
				min := strconv.FormatUint(uint64(uint32Slice[0]), 10)
				max := strconv.FormatUint(uint64(uint32Slice[1]), 10)
				res, err := rCli.ZRangeByScore(ctx, CHANNEL, &redis.ZRangeBy{Min: min, Max: max}).Result()
				if err != nil {
					fmt.Println(err)
				}

				uint8SliceLen := uint32Slice[1] - uint32Slice[0] + 1 + 4
				uint8Slice := make([]uint8, uint8SliceLen)

				for i := 0; i < 4; i++ {
					uint8Slice[i] = data[i]
				}

				for i := 0; i < len(res); i++ {
					strValue := res[i]

					uint64value, err := strconv.ParseUint(strValue, 10, 32)
					if err != nil {
						fmt.Println(err)
					}

					uint32Value := uint32(uint64value)
					bitIndex := uint32Value - uint32Slice[0]

					byteIndex := bitIndex/8 + 4

					uint8Slice[byteIndex] |= (1 << (bitIndex % 8))
				}

				fmt.Println(data)
				fmt.Println(uint32Slice)
				fmt.Println(res)
				fmt.Println(uint8Slice, len(uint8Slice), uint8Slice[4])

				mutex.Lock()
				err = con.WriteMessage(websocket.BinaryMessage, uint8Slice)
				mutex.Unlock()
				if err != nil {
					fmt.Println(err)
				}
			}

		}
	}()

	for {
		select {
		case msg := <-pubsub.Channel():

			uint32Value := binary.LittleEndian.Uint32([]byte(msg.Payload))

			buf := make([]byte, 4)

			binary.LittleEndian.PutUint32(buf, uint32Value)

			mutex.Lock()
			err = con.WriteMessage(websocket.BinaryMessage, buf)
			mutex.Unlock()
			if err != nil {
				log.Printf("[websockets error]: error sending message to client: %s\n", err.Error())
				return
			}

		case <-ticker.C:
			mutex.Lock()
			err := con.WriteMessage(websocket.PingMessage, nil)
			mutex.Unlock()
			if err != nil {
				log.Printf("[websockets error]: error sending ping message to client: %s\n", err.Error())
				return
			}

		case <-ctx.Done():
			return
		}
	}
}
