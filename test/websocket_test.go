package main

import (
	"encoding/binary"
	"math/rand/v2"
	"testing"

	"github.com/gorilla/websocket"
)

func BenchmarkWebSocket(b *testing.B) {
	for i := 0; i < b.N; i++ {
		con, _, err := websocket.DefaultDialer.Dial(envData.WEBSOCKET_URL, nil)
		if err != nil {
			b.Fatal(err)
		}

		uint32Value := rand.Uint32()%TOTAL_CHECKBOXES + 1

		uint8Slice := make([]uint8, 4)
		binary.LittleEndian.PutUint32(uint8Slice, uint32Value)

		if err := con.WriteMessage(websocket.BinaryMessage, uint8Slice); err != nil {
			b.Fatal(err)
		}

		_, _, err = con.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}

		con.Close()
	}
}
