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


func Dial(host string, timeout time.Duration) (*Socket, error) {
    dialer := &websocket.Dialer{HandshakeTimeout: timeout}
    u, err := url.Parse(host)
    if err != nil {
        return nil, err
    }

    u.Path = fmt.Sprintf("/socket.io/%d/", 1)
    resp, err := http.Get(u.String())
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return nil, err
    }

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    parts := strings.SplitN(string(body), ":", 4)
    if len(parts) != 4 {
        return nil, errors.New("invalid handshake: " + string(body))
    }

    if !strings.Contains(parts[3], "websocket") {
        return nil, errors.New("server does not support websockets")
    }

    sessionId := parts[0]
    u.Scheme = "ws" + u.Scheme[4:]
    u.Path = fmt.Sprintf("%swebsocket/%s", u.Path, sessionId)

    io := &Socket{}
    io.timeout = timeout
    io.heartbeat = 10 * time.Second
    io.conn, _, err = dialer.Dial(u.String(), nil)
    if err != nil {
        return nil, err
    }

    go io.readLoop()

    go func() {
        for _ = range time.Tick(io.heartbeat) {
            log.Println("hb")
            if err := io.sendHeartbeat(); err != nil {
                log.Println(err, "vvvv")
                return
            }
        }
    }()

    return io, nil
}

func (io *Socket) Emit(name string, data ...interface{}) error {
    b, err := json.Marshal(data)
    if err != nil {
        return err
    }
    raw := fmt.Sprintf(`5:::{"name":"%s","args":%s}`, name, string(b))
    return io.conn.WriteMessage(websocket.TextMessage, []byte(raw))
}

func (io *Socket) EmitText(name string, raw string) error {
    r := fmt.Sprintf(`5:::{"name":"%s","args":[%s]}`, name, raw)
    log.Println(r)
    return io.conn.WriteMessage(websocket.TextMessage, []byte(r))
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
        log.Println(msg.Type, string(msg.Data), err)
        if err != nil {
            continue
        }
    }
}
