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

type messageDTO struct {
	messageType int
	message     []uint8
}

type websocketHandler struct {
	connection *websocket.Conn
	send       chan *messageDTO
	done       chan struct{}
	once       *sync.Once
}

func handleWebsocketConnection(connection *websocket.Conn) {
	send := make(chan *messageDTO, BUFFERS_SIZE)
	done := make(chan struct{})
	once := &sync.Once{}

	h := &websocketHandler{
		connection: connection,
		send:       send,
		done:       done,
		once:       once,
	}

	go h.listener()
	go h.reader()
	go h.writer()

	h.connection.SetPingHandler(func(msg string) error {
		h.queue(&messageDTO{messageType: websocket.PongMessage, message: []uint8(msg)})
		return nil
	})

	h.connection.SetReadDeadline(time.Now().Add(READ_TIMEOUT))

	h.connection.SetPongHandler(func(string) error {
		h.connection.SetReadDeadline(time.Now().Add(READ_TIMEOUT))
		return nil
	})

	uint8Slice := make([]uint8, 4)

	checks, err := rCli.ZCard(context.Background(), REDIS_KEY).Result()
	if err != nil {
		log.Printf("[redis error]: error getting checks: %s\n", err.Error())
		return
	}
	uint32Checks := uint32(checks)
	binary.LittleEndian.PutUint32(uint8Slice, uint32Checks)

	h.queue(&messageDTO{messageType: websocket.BinaryMessage, message: uint8Slice})
}

func (h *websocketHandler) close() {
	h.once.Do(func() {
		close(h.done)
		h.send = nil
		h.connection.Close()
	})
}

func (h *websocketHandler) reader() {
	defer h.close()

	for {
		_, message, err := h.connection.ReadMessage()
		if err != nil {
			log.Printf("[websockets error]: error reading message from client: %s\n", err.Error())
			return
		}

		messageLen := len(message)

		if messageLen != 4 && messageLen != 8 {
			continue
		}

		uint32SliceLen := messageLen / 4
		uint32Slice := make([]uint32, uint32SliceLen)

		for i := 0; i < uint32SliceLen; i++ {
			start := i * 4
			uint32Value := binary.LittleEndian.Uint32(message[start:])
			uint32Slice[i] = uint32Value
		}

		if uint32SliceLen == 1 {
			if uint32Slice[0] < 1 || uint32Slice[0] > TOTAL_CHECKBOXES {
				continue
			}

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
				uint8Slice[i+4] = message[i]
			}

			if err := rCli.Publish(context.Background(), REDIS_CHANNEL, uint8Slice).Err(); err != nil {
				log.Printf("[redis error]: error publishing checkbox: %s\n", err.Error())
			}
		} else { // uint32SliceLen == 2
			if uint32Slice[0] < 1 || uint32Slice[0] > TOTAL_CHECKBOXES || uint32Slice[1] < uint32Slice[0] || uint32Slice[1] > TOTAL_CHECKBOXES {
				continue
			}

			min := strconv.FormatUint(uint64(uint32Slice[0]), 10)
			max := strconv.FormatUint(uint64(uint32Slice[1]), 10)
			checkboxes, err := rCli.ZRangeByScore(context.Background(), REDIS_KEY, &redis.ZRangeBy{Min: min, Max: max}).Result()
			if err != nil {
				log.Printf("[redis error]: error getting checked checkboxes: %s\n", err.Error())
				continue
			}

			uint8SliceLen := uint32Slice[1] - uint32Slice[0] + 1 + 4
			uint8Slice := make([]uint8, uint8SliceLen)

			for i := 0; i < 4; i++ {
				uint8Slice[i] = message[i]
			}

			for i := 0; i < len(checkboxes); i++ {
				strValue := checkboxes[i]
				uint64value, err := strconv.ParseUint(strValue, 10, 32)
				if err != nil {
					continue
				}
				uint32Value := uint32(uint64value)

				bitIndex := uint32Value - uint32Slice[0]
				byteIndex := bitIndex / 8
				uint8Slice[byteIndex+4] |= (1 << (bitIndex % 8))
			}

			h.queue(&messageDTO{messageType: websocket.BinaryMessage, message: uint8Slice})
		}
	}
}

func (h *websocketHandler) listener() {
	defer h.close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pubsub := rCli.Subscribe(ctx, REDIS_CHANNEL)
	defer pubsub.Close()

	for {
		select {
		case message, ok := <-pubsub.Channel(redis.WithChannelSize(BUFFERS_SIZE), redis.WithChannelSendTimeout(WRITE_TIMEOUT)):
			if !ok {
				return
			}
			h.queue(&messageDTO{messageType: websocket.BinaryMessage, message: []uint8(message.Payload)})

		case <-h.done:
			return
		}
	}
}

func (h *websocketHandler) queue(message *messageDTO) {
	timer := time.NewTimer(WRITE_TIMEOUT)
	defer timer.Stop()

	select {
	case h.send <- message:
	case <-timer.C:
	case <-h.done:
	}
}

func (h *websocketHandler) writer() {
	defer h.close()

	ticker := time.NewTicker(PING_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case message := <-h.send:
			h.connection.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))

			if err := h.connection.WriteMessage(message.messageType, message.message); err != nil {
				log.Printf("[websockets error]: error sending message to client: %s\n", err.Error())
				return
			}

		case <-ticker.C:
			h.connection.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))

			if err := h.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[websockets error]: error sending ping message to client: %s\n", err.Error())
				return
			}

		case <-h.done:
			return
		}
	}
}
