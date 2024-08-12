package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type connectionsPool struct {
	connectionsHandlers map[*connectionHandler]struct{}
	mutex               *sync.Mutex
}

func newConnectionsPool() *connectionsPool {
	return &connectionsPool{connectionsHandlers: make(map[*connectionHandler]struct{}), mutex: &sync.Mutex{}}
}

func (h *connectionsPool) Add(connectionHandler *connectionHandler) {
	h.mutex.Lock()
	h.connectionsHandlers[connectionHandler] = struct{}{}
	h.mutex.Unlock()
}

func (h *connectionsPool) Remove(connectionHandler *connectionHandler) {
	h.mutex.Lock()
	delete(h.connectionsHandlers, connectionHandler)
	h.mutex.Unlock()
}

func (h *connectionsPool) Close() {
	for connectionHandler := range h.connectionsHandlers {
		connectionHandler.send <- message{messageType: websocket.CloseMessage, message: websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "")}
		connectionHandler.close()
	}
}
