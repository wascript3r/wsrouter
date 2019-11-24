package websocket

import (
	"io"

	"github.com/gobwas/ws"
)

type Request struct {
	Op  ws.OpCode
	Req io.Reader
}

func NewRequest(op ws.OpCode, r io.Reader) Request {
	return Request{op, r}
}
