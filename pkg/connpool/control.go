package connpool

import (
	"encoding/json"
	"fmt"

	"github.com/telepresenceio/telepresence/rpc/v2/manager"
)

type ControlCode byte

const (
	SessionInfo = ControlCode(iota)
	Connect
	ConnectOK
	ConnectReject
	Disconnect
	DisconnectOK
	ReadClosed
	WriteClosed
	KeepAlive
)

func (c ControlCode) String() string {
	switch c {
	case SessionInfo:
		return "SESSION_ID"
	case Connect:
		return "CONNECT"
	case ConnectOK:
		return "CONNECT_OK"
	case ConnectReject:
		return "CONNECT_REJECT"
	case Disconnect:
		return "DISCONNECT"
	case DisconnectOK:
		return "DISCONNECT_OK"
	case ReadClosed:
		return "READ_CLOSED"
	case WriteClosed:
		return "WRITE_CLOSED"
	case KeepAlive:
		return "KEEP_ALIVE"
	default:
		return fmt.Sprintf("** unknown control code: %d **", c)
	}
}

type Control interface {
	Message
	Code() ControlCode
	SessionInfo() *manager.SessionInfo
}

type control struct {
	code    ControlCode
	id      ConnID
	payload []byte
}

func (c *control) Code() ControlCode {
	return c.code
}

func (c *control) ID() ConnID {
	return c.id
}

func (c *control) Payload() []byte {
	return c.payload
}

// SessionInfo returns the SessionInfo that this Control represents or nil if
// this isn't a SessionInfo Control.
func (c *control) SessionInfo() *manager.SessionInfo {
	if c.code == SessionInfo {
		var sessionInfo *manager.SessionInfo
		if err := json.Unmarshal(c.payload, &sessionInfo); err == nil {
			return sessionInfo
		}
	}
	return nil
}

func (c *control) String() string {
	return fmt.Sprintf("%s, code %s, len %d", c.id, c.code, len(c.payload))
}

func (c *control) TunnelMessage() *manager.ConnMessage {
	idLen := len(c.id)
	cmPl := make([]byte, idLen+len(c.payload))
	copy(cmPl, c.id)
	copy(cmPl[idLen:], c.payload)
	return &manager.ConnMessage{ConnId: []byte{byte(c.code), byte(idLen)}, Payload: cmPl}
}

func NewControl(id ConnID, code ControlCode, payload []byte) Control {
	return &control{id: id, code: code, payload: payload}
}

func SessionInfoControl(sessionInfo *manager.SessionInfo) Control {
	jsonInfo, err := json.Marshal(sessionInfo)
	if err != nil {
		// The SessionInfo must be json Marshable
		panic(err)
	}
	return &control{id: "", code: SessionInfo, payload: jsonInfo}
}
