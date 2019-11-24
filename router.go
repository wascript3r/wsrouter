package wsrouter

import (
	"errors"

	"github.com/wascript3r/websocket"
)

var (
	ErrMethodNotFound = errors.New("method not found")
)

type Handler func(*websocket.Socket, *Command)

type Router struct {
	handler map[string]Handler
}

func New() *Router {
	return &Router{
		make(map[string]Handler),
	}
}

func (r *Router) Register(m string, h Handler) {
	r.handler[m] = h
}

func (r *Router) Handle(s *websocket.Socket, cmd *Command) error {
	hnd, ok := r.handler[cmd.Method]
	if !ok {
		return ErrMethodNotFound
	}
	hnd(s, cmd)
	return nil
}
