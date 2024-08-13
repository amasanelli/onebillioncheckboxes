package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type handlersPool struct {
	handlers map[*connectionHandler]struct{}
	mutex    *sync.Mutex
}

func newHandlersPool() *handlersPool {
	return &handlersPool{handlers: make(map[*connectionHandler]struct{}), mutex: &sync.Mutex{}}
}

func (p *handlersPool) Add(handler *connectionHandler) {
	p.mutex.Lock()
	p.handlers[handler] = struct{}{}
	p.mutex.Unlock()
}

func (p *handlersPool) Remove(handler *connectionHandler) {
	p.mutex.Lock()
	delete(p.handlers, handler)
	p.mutex.Unlock()
}

func (p *handlersPool) Close() {
	for handler := range p.handlers {
		handler.send <- message{messageType: websocket.CloseMessage, message: websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "")}
		handler.close()
	}
}
