package wsrouter

import (
	"github.com/wascript3r/websocket"
)

type Response struct {
	Error  *string `json:"e"`
	Params Object  `json:"p"`
}

func WriteErrorRes(s *websocket.Socket, e string) error {
	return s.WriteDataJSON(Response{
		Error:  &e,
		Params: nil,
	})
}

func WriteSuccessRes(s *websocket.Socket, p Object) error {
	return s.WriteDataJSON(Response{
		Error:  nil,
		Params: p,
	})
}
