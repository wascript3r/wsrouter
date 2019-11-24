package wsrouter

import (
	"encoding/json"
	"io"
)

type Command struct {
	Method string `json:"m"`
	Params Object `json:"p"`
}

func ParseCommand(r io.Reader) (*Command, error) {
	cmd := &Command{}
	dec := json.NewDecoder(r)

	if err := dec.Decode(cmd); err != nil {
		return nil, err
	}

	return cmd, nil
}
