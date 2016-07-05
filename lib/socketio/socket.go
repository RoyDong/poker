package socketio

import (
    "time"
    "net/url"
    "fmt"
    "net/http"
    "io/ioutil"
    "strings"
    "errors"
    "encoding/json"
    "log"
    "github.com/gorilla/websocket"
)

type Socket struct {
    conn *websocket.Conn
    timeout time.Duration
    heartbeat time.Duration
}

func NewSocket(timeout time.Duration) *Socket {
    io := &Socket{}
    io.timeout = timeout
    io.heartbeat = 10
    return io
}

func (io *Socket) Dial(host string) error {
    dialer := &websocket.Dialer{HandshakeTimeout: io.timeout}
    u, err := url.Parse(host)
    if err != nil {
        return err
    }

    u.Path = fmt.Sprintf("/socket.io/%d/", 1)
    resp, err := http.Get(u.String())
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return err
    }

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    parts := strings.SplitN(string(body), ":", 4)
    if len(parts) != 4 {
        return errors.New("invalid handshake: " + string(body))
    }

    if !strings.Contains(parts[3], "websocket") {
        return errors.New("server does not support websockets")
    }

    sessionId := parts[0]
    u.Scheme = "ws" + u.Scheme[4:]
    u.Path = fmt.Sprintf("%swebsocket/%s", u.Path, sessionId)

    io.conn, _, err = dialer.Dial(u.String(), nil)
    if err != nil {
        return err
    }

    go io.readLoop()

    go func() {
        for _ = range time.Tick(io.heartbeat) {
            if err := io.sendHeartbeat(); err != nil {
                return
            }
        }
    }()

    return nil
}

func (io *Socket) Emit(name string, data ...interface{}) error {
    b, err := json.Marshal(data)
    if err != nil {
        return err
    }
    raw := fmt.Sprintf(`5:::{"name":"%s","args":%s}`, name, string(b))
    return io.conn.WriteMessage(websocket.TextMessage, []byte(raw))
}

func (io *Socket) sendConnect() error {
    return io.conn.WriteMessage(websocket.TextMessage, []byte("1::"))
}

func (io *Socket) sendHeartbeat() error {
    return io.conn.WriteMessage(websocket.TextMessage, []byte("2::"))
}

func (io *Socket) sendAck() error {
    return io.conn.WriteMessage(websocket.TextMessage, []byte("6::"))
}

func (io *Socket) Read() ([]byte, error) {
    _, raw, err := io.conn.ReadMessage()
    return raw, err
}

func (io *Socket) readLoop() {
    for {
        _, raw, err := io.conn.ReadMessage()
        if err != nil {
            return
        }
        msg, err := parseMessage(raw)
        if err != nil {
            continue
        }
        log.Println(msg.ID, msg.Type)
    }
}
