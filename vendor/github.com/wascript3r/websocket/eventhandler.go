package websocket

import "net"

type Resume func()

type EventHandler interface {
	OnOpen(*Socket)
	OnDisconnect(*Socket)
	OnRead(*Socket, Request, Resume)
	OnError(err error, conn net.Conn)
}
