package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type handlersPool struct {
	handlers map[*websocketHandler]struct{}
	mutex    *sync.Mutex
}

func newHandlersPool() *handlersPool {
	return &handlersPool{handlers: make(map[*websocketHandler]struct{}), mutex: &sync.Mutex{}}
}

func (p *handlersPool) add(handler *websocketHandler) {
	p.mutex.Lock()
	p.handlers[handler] = struct{}{}
	p.mutex.Unlock()
}

func (p *handlersPool) remove(handler *websocketHandler) {
	p.mutex.Lock()
	delete(p.handlers, handler)
	p.mutex.Unlock()
}

func (p *handlersPool) close() {
	for handler := range p.handlers {
		handler.send <- messageDTO{messageType: websocket.CloseMessage, message: websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "")}
		handler.close()
	}
}
