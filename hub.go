package main

import (
	"context"
	"encoding/binary"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type message struct {
	messageType int
	message     []uint8
}

type connectionHandler struct {
	connection *websocket.Conn
	send       chan message
	done       chan struct{}
	once       *sync.Once
}

func newConnectionHandler(connection *websocket.Conn) *connectionHandler {
	send := make(chan message, 256)
	done := make(chan struct{})
	once := &sync.Once{}

	connection.SetPingHandler(func(msg string) error {
		message := message{
			messageType: websocket.PongMessage,
			message:     []uint8(msg),
		}
		send <- message
		return nil
	})

	connection.SetWriteDeadline(time.Now().Add(writeTimeout))

	connection.SetPongHandler(func(string) error {
		connection.SetWriteDeadline(time.Now().Add(writeTimeout))
		return nil
	})

	connectionHandler := &connectionHandler{
		connection: connection,
		send:       send,
		done:       done,
		once:       once,
	}

	pool.Add(connectionHandler)

	return connectionHandler
}

func (h *connectionHandler) run() {
	go h.listen()
	go h.read()
	go h.write()

	uint8Slice := make([]uint8, 4)

	checks, err := rCli.ZCard(context.Background(), REDIS_KEY).Result()
	if err != nil {
		log.Printf("[redis error]: error getting checks: %s\n", err.Error())
		return
	}
	uint32Checks := uint32(checks)
	binary.LittleEndian.PutUint32(uint8Slice, uint32Checks)

	h.send <- message{messageType: websocket.BinaryMessage, message: uint8Slice}
}

func (h *connectionHandler) close() {
	h.once.Do(func() {
		close(h.done)
		pool.Remove(h)
		h.connection.Close()
	})
}

func (h *connectionHandler) read() {
	defer h.close()

	for {
		_, msg, err := h.connection.ReadMessage()
		if err != nil {
			log.Printf("[websockets error]: error reading message from client: %s\n", err.Error())
			return
		}

		messageLen := len(msg)

		if messageLen != 4 && messageLen != 8 {
			continue
		}

		uint32SliceLen := messageLen / 4
		uint32Slice := make([]uint32, uint32SliceLen)

		for i := 0; i < uint32SliceLen; i++ {
			start := i * 4
			uint32Slice[i] = binary.LittleEndian.Uint32(msg[start:])
		}

		if uint32SliceLen == 1 {
			member := strconv.FormatUint(uint64(uint32Slice[0]), 10)
			score := float64(uint32Slice[0])
			if err := rCli.ZAdd(context.Background(), REDIS_KEY, redis.Z{Score: score, Member: member}).Err(); err != nil {
				log.Printf("[redis error]: error storing checkbox: %s\n", err.Error())
				continue
			}

			uint8Slice := make([]uint8, 8)

			checks, err := rCli.ZCard(context.Background(), REDIS_KEY).Result()
			if err != nil {
				log.Printf("[redis error]: error getting checks: %s\n", err.Error())
				continue
			}
			uint32Checks := uint32(checks)
			binary.LittleEndian.PutUint32(uint8Slice[:4], uint32Checks)

			for i := 0; i < 4; i++ {
				uint8Slice[i+4] = msg[i]
			}

			if err := rCli.Publish(context.Background(), REDIS_CHANNEL, uint8Slice).Err(); err != nil {
				log.Printf("[redis error]: error publishing checkbox: %s\n", err.Error())
			}

			continue
		}

		if uint32SliceLen == 2 {
			min := strconv.FormatUint(uint64(uint32Slice[0]), 10)
			max := strconv.FormatUint(uint64(uint32Slice[1]), 10)
			checkboxes, err := rCli.ZRangeByScore(context.Background(), REDIS_KEY, &redis.ZRangeBy{Min: min, Max: max}).Result()
			if err != nil {
				log.Printf("[redis error]: error getting checkboxes: %s\n", err.Error())
				continue
			}

			uint8SliceLen := uint32Slice[1] - uint32Slice[0] + 1 + 4
			uint8Slice := make([]uint8, uint8SliceLen)

			for i := 0; i < 4; i++ {
				uint8Slice[i] = msg[i]
			}

			for i := 0; i < len(checkboxes); i++ {
				strValue := checkboxes[i]
				uint64value, err := strconv.ParseUint(strValue, 10, 32)
				if err != nil {
					log.Printf("[internal server error]: error parsing checkboxes data: %s\n", err.Error())
					continue
				}
				uint32Value := uint32(uint64value)

				bitIndex := uint32Value - uint32Slice[0]
				byteIndex := bitIndex / 8
				uint8Slice[byteIndex+4] |= (1 << (bitIndex % 8))
			}

			h.send <- message{messageType: websocket.BinaryMessage, message: uint8Slice}

			continue
		}
	}
}

func (h *connectionHandler) listen() {
	defer h.close()

	pubsub := rCli.Subscribe(context.Background(), REDIS_CHANNEL)
	defer pubsub.Close()

	for {
		select {
		case msg, ok := <-pubsub.Channel():
			if !ok {
				return
			}
			h.send <- message{messageType: websocket.BinaryMessage, message: []uint8(msg.Payload)}

		case <-h.done:
			return
		}
	}
}

func (h *connectionHandler) write() {
	defer h.close()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case message := <-h.send:
			if err := h.connection.WriteMessage(message.messageType, message.message); err != nil {
				log.Printf("[websockets error]: error sending message to client: %s\n", err.Error())
				return
			}

		case <-ticker.C:
			if err := h.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[websockets error]: error sending ping message to client: %s\n", err.Error())
				return
			}

		case <-h.done:
			return
		}
	}
}
