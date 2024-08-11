package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type connectionPool struct {
	conns map[*websocket.Conn]struct{}
	mutex *sync.Mutex
}

func newPool() *connectionPool {
	return &connectionPool{conns: make(map[*websocket.Conn]struct{}), mutex: &sync.Mutex{}}
}

func (h *connectionPool) Add(con *websocket.Conn) {
	h.mutex.Lock()
	h.conns[con] = struct{}{}
	h.mutex.Unlock()
}

func (h *connectionPool) Remove(con *websocket.Conn) {
	h.mutex.Lock()
	delete(h.conns, con)
	h.mutex.Unlock()
}

func (h *connectionPool) Close() {
	for con := range h.conns {
		con.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, ""))
		con.Close()
	}
}
