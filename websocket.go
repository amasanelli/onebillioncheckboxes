package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
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
	limiter    *rate.Limiter
}

func handleWebsocketConnection(connection *websocket.Conn) {
	send := make(chan *messageDTO, BUFFERS_SIZE)
	done := make(chan struct{})
	once := &sync.Once{}
	limiter := rate.NewLimiter(rate.Limit(envData.LIMITER_LIMIT), envData.LIMITER_BURST)

	h := &websocketHandler{
		connection: connection,
		send:       send,
		done:       done,
		once:       once,
		limiter:    limiter,
	}

	go h.listener()
	go h.reader()
	go h.writer()

	h.connection.SetPingHandler(func(message string) error {
		h.queue(&messageDTO{messageType: websocket.PongMessage, message: []uint8(message)})
		return nil
	})

	h.connection.SetReadDeadline(time.Now().Add(READ_TIMEOUT))

	h.connection.SetPongHandler(func(string) error {
		h.connection.SetReadDeadline(time.Now().Add(READ_TIMEOUT))
		return nil
	})

	strChecks, err := rCli.Get(context.Background(), REDIS_CHECKS_COUNT_KEY).Result()
	if err != nil {
		log.Printf("[redis error]: error getting checks count: %s\n", err.Error())
		return
	}
	int64Checks, err := strconv.ParseInt(strChecks, 10, 64)
	if err != nil {
		log.Printf("[redis error]: error parsing checks count: %s\n", err.Error())
		return
	}
	uint32Checks := uint32(int64Checks)

	uint8Slice := make([]uint8, 4)

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

		if !h.limiter.Allow() {
			return
		}

		messageLen := len(message)

		if messageLen != 4 && messageLen != 8 {
			return
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
				return
			}

			strCheckbox := strconv.FormatUint(uint64(uint32Slice[0]), 10)
			float64Checkbox := float64(uint32Slice[0])
			keyIndex := (uint32Slice[0] - 1) / REDIS_CHECKS_PER_KEY
			key := fmt.Sprintf("%s_%d", REDIS_CHECKS_KEY, keyIndex)

			newMembers, err := rCli.ZAdd(context.Background(), key, redis.Z{Score: float64Checkbox, Member: strCheckbox}).Result()
			if err != nil {
				log.Printf("[redis error]: error storing checkbox: %s\n", err.Error())
				continue
			}

			if newMembers == 0 {
				return
			}

			// if this fails, the counter gets inconsistent
			int64Checks, err := rCli.Incr(context.Background(), REDIS_CHECKS_COUNT_KEY).Result()
			if err != nil {
				log.Printf("[redis error]: error getting checks count: %s\n", err.Error())
				continue
			}
			uint32Checks := uint32(int64Checks)

			uint8Slice := make([]uint8, 8)

			binary.LittleEndian.PutUint32(uint8Slice[:4], uint32Checks)

			for i := 0; i < 4; i++ {
				uint8Slice[i+4] = message[i]
			}

			if err := rCli.Publish(context.Background(), REDIS_CHANNEL, uint8Slice).Err(); err != nil {
				log.Printf("[redis error]: error publishing checkbox: %s\n", err.Error())
				continue
			}
		} else { // uint32SliceLen == 2
			if uint32Slice[0] < 1 || uint32Slice[0] > TOTAL_CHECKBOXES || uint32Slice[1] < uint32Slice[0] || uint32Slice[1] > TOTAL_CHECKBOXES {
				return
			}

			checkboxes := uint32Slice[1] - uint32Slice[0] + 1

			if checkboxes > MAX_CHECKBOXES_PER_REQUEST {
				return
			}

			minCheckbox := strconv.FormatUint(uint64(uint32Slice[0]), 10)
			maxCheckbox := strconv.FormatUint(uint64(uint32Slice[1]), 10)

			keyIndexMin := (uint32Slice[0] - 1) / REDIS_CHECKS_PER_KEY
			keyIndexMax := (uint32Slice[1] - 1) / REDIS_CHECKS_PER_KEY

			keyMin := fmt.Sprintf("%s_%d", REDIS_CHECKS_KEY, keyIndexMin)
			strSliceCheckboxes, err := rCli.ZRangeByScore(context.Background(), keyMin, &redis.ZRangeBy{Min: minCheckbox, Max: maxCheckbox}).Result()
			if err != nil {
				log.Printf("[redis error]: error getting checked checkboxes: %s\n", err.Error())
				continue
			}

			if keyIndexMin != keyIndexMax {
				keyMax := fmt.Sprintf("%s_%d", REDIS_CHECKS_KEY, keyIndexMax)
				strSliceCheckboxesMax, err := rCli.ZRangeByScore(context.Background(), keyMax, &redis.ZRangeBy{Min: minCheckbox, Max: maxCheckbox}).Result()
				if err != nil {
					log.Printf("[redis error]: error getting checked checkboxes: %s\n", err.Error())
					continue
				}
				strSliceCheckboxes = append(strSliceCheckboxes, strSliceCheckboxesMax...)
			}

			uint8SliceLen := (checkboxes+7)/8 + 4
			uint8Slice := make([]uint8, uint8SliceLen)

			for i := 0; i < 4; i++ {
				uint8Slice[i] = message[i]
			}

			for i := 0; i < len(strSliceCheckboxes); i++ {
				strCheckbox := strSliceCheckboxes[i]
				uint64Checkbox, err := strconv.ParseUint(strCheckbox, 10, 64)
				if err != nil {
					log.Printf("[redis error]: error parsing checked checkboxes: %s\n", err.Error())
					continue
				}
				uint32Checkbox := uint32(uint64Checkbox)

				bitIndex := uint32Checkbox - uint32Slice[0]
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
