package socketio

import (
    "errors"
    "strconv"
    "fmt"
    "bytes"
    "github.com/roydong/gmvc"
)

type Message struct {
    Type     int
    ID       string
    Endpoint string
    Data     []byte
}

// ParseMessages parses the given raw message in to Message.
func parseMessage(raw []byte) (*Message, error) {
    parts := bytes.SplitN(raw, []byte(":"), 4)
    if len(parts) < 3 {
        return nil, errors.New("Empty message")
    }

    msgType, err := strconv.Atoi(string(parts[0]))
    if err != nil {
        return nil, err
    }

    msg := &Message{msgType, string(parts[1]), string(parts[2]), nil}
    if len(parts) == 4 {
        msg.Data = parts[3]
    }

    return msg, nil
}

// String returns the string represenation of the Message.
func (m *Message) String() string {
    raw := fmt.Sprintf("%d:%s:%s", m.Type, m.ID, m.Endpoint)
    if len(m.Data) > 0 {
        raw = fmt.Sprintf("%s:%s", raw, m.Data)
    }
    return raw
}

func (m *Message) ReadData() *gmvc.Tree {
    tree := gmvc.NewTree()
    err := tree.LoadJson("", m.Data, true)
    if err != nil {
        return nil
    }
    return tree
}

// NewDisconnect returns a new disconnect Message.
func NewDisconnect() *Message {
    return &Message{Type: 0}
}

// NewConnect returns a new connect Message for the given endpoint.
func NewConnect(endpoint string) *Message {
    return &Message{Type: 1, Endpoint: endpoint}
}

// NewHeartbeat returns a new heartbeat Message.
func NewHeartbeat() *Message {
    return &Message{Type: 2}
}

func NewMessageMsg(endpoint string, msg []byte) *Message {
    return &Message{Type: 3, Endpoint: endpoint, Data: msg}
}

func NewJSONMessage(endpoint string, data []byte) *Message {
    return &Message{Type: 4, Endpoint: endpoint, Data: data}
}

func NewEvent(endpoint string, name string, args []byte) *Message {
    return &Message{Type: 5, Endpoint: endpoint, Data: args}
}

func NewACK(data []byte) *Message {
    return &Message{Type: 6, Data: data}
}

// NewError returns a new error Message for the given endpoint with a
// reason and an advice.
func NewError(endpoint, msg string) *Message {
    return &Message{Type: 7, Endpoint: endpoint, Data: []byte(msg)}
}

// NewNoop returns a new no-op Message.
func NewNoop() *Message {
    return &Message{Type: 8}
}


