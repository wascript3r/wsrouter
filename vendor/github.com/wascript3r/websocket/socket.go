package websocket

import (
	"encoding/json"
	"io"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type Socket struct {
	io sync.Mutex
	net.Conn
}

func NewSocket(conn net.Conn) *Socket {
	return &Socket{
		sync.Mutex{},
		conn,
	}
}

func (s *Socket) getReaderUnsafe() (ws.OpCode, io.Reader, error) {
	h, r, err := wsutil.NextReader(s, ws.StateServerSide)
	if err != nil {
		return 0, nil, err
	}
	if h.OpCode.IsControl() {
		return h.OpCode, nil, wsutil.ControlFrameHandler(s, ws.StateServerSide)(h, r)
	}

	return h.OpCode, r, nil
}

func (s *Socket) ReadDataJSON(v interface{}) error {
	s.io.Lock()
	defer s.io.Unlock()

	_, r, err := s.getReaderUnsafe()
	if err != nil {
		return err
	}

	d := json.NewDecoder(r)
	return d.Decode(v)
}

func (s *Socket) getWriterUnsafe(op ws.OpCode) *wsutil.Writer {
	return wsutil.NewWriter(s, ws.StateServerSide, op)
}

func (s *Socket) WriteData(p []byte) error {
	return wsutil.WriteServerText(s, p)
}

func (s *Socket) WriteDataJSON(v interface{}) error {
	w := s.getWriterUnsafe(ws.OpText)
	enc := json.NewEncoder(w)

	s.io.Lock()
	defer s.io.Unlock()

	if err := enc.Encode(v); err != nil {
		return err
	}

	return w.Flush()
}

func (s *Socket) writeCloseMessage(reason string) error {
	s.io.Lock()
	defer s.io.Unlock()

	return wsutil.WriteServerMessage(s, ws.OpClose, ws.NewCloseFrameBody(1000, reason))
}

func (s *Socket) Disconnect(reason string) error {
	err := s.writeCloseMessage(reason)
	if err != nil {
		return err
	}

	return s.Close()
}

func (s *Socket) ForceDisconnect(reason string) {
	s.writeCloseMessage(reason)
	s.Close()
}
