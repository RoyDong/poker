package market

import (
    "net/http"
    "net/url"
    "fmt"
    "strings"
    "sort"
    "github.com/roydong/gmvc"
    "github.com/gorilla/websocket"
    "io/ioutil"
    "time"
    "errors"
    "encoding/json"
    "log"
)

func CallRest(api string, query, data map[string]interface{}) *gmvc.Tree {
    if query != nil {
        api = api + "?" + BuildHttpQuery(query)
    }

    var resp *http.Response
    var err error
    if data == nil {
         resp, err = http.Get(api)
    } else {
        form := url.Values{}
        for k, v := range data {
            form.Set(k, fmt.Sprintf("%v", v))
        }
        resp, err = http.PostForm(api, form)
    }
    if err != nil {
        gmvc.Logger.Println("call " + api + " error")
        return nil
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        gmvc.Logger.Println("call " + api + " error")
        return nil
    }

    tree := gmvc.NewTree()
    if err = tree.LoadJson("", body, false); err != nil {
        gmvc.Logger.Println("call " + api + " error not json")
        return nil
    }

    return tree
}

func BuildHttpQuery(data map[string]interface{}) string {
    query := make([]string, 0, len(data))
    for k, v := range data {
        query = append(query, fmt.Sprintf("%s=%v", k, v))
    }
    return strings.Join(query, "&")
}



func createSignature(params map[string]interface{}, skey string) string {
    keys := make([]string, 0, len(params))
    for k := range params {
        keys = append(keys, k)
    }

    sort.Strings(keys)

    sigs := make([]string, 0, len(keys))
    for _, k := range keys {
        sigs = append(sigs, fmt.Sprintf("%s=%v", k, params[k]))
    }

    sigs = append(sigs, "secret_key=" + skey)

    return gmvc.MD5(strings.Join(sigs, "&"))
}


type SocketIO struct {
    conn *websocket.Conn
    timeout time.Duration
    heartbeat time.Duration
}

func NewSocketIO(timeout time.Duration) *SocketIO {
    io := &SocketIO{}
    io.timeout = timeout
    io.heartbeat = 10
    return io
}

func (io *SocketIO) Dial(host string) error {
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

    go io.listen()

    go func() {
        for _ = range time.Tick(io.heartbeat) {
            if err := io.sendHeartbeat(); err != nil {
                return
            }
        }
    }()

    return nil
}

func (io *SocketIO) Emit(name string, data ...interface{}) error {
    b, err := json.Marshal(data)
    if err != nil {
        return err
    }
    raw := fmt.Sprintf(`5:::{"name":"%s","args":%s}`, name, string(b))
    return io.conn.WriteMessage(websocket.TextMessage, []byte(raw))
}

func (io *SocketIO) sendConnect() error {
    return io.conn.WriteMessage(websocket.TextMessage, []byte("1::"))
}

func (io *SocketIO) sendHeartbeat() error {
    return io.conn.WriteMessage(websocket.TextMessage, []byte("2::"))
}

func (io *SocketIO) sendAck() error {
    return io.conn.WriteMessage(websocket.TextMessage, []byte("6::"))
}

func (io *SocketIO) Read() ([]byte, error) {
    _, raw, err := io.conn.ReadMessage()
    return raw, err
}

func (io *SocketIO) listen() {
    for {
        _, raw, err := io.conn.ReadMessage()
        if err != nil {
            return
        }
        log.Println(string(raw))
        //io.parseMessage(string(raw))
    }
}

// ParseMessages parses the given raw message in to Message.
func (io *SocketIO) parseMessage(raw string) {
    parts := strings.SplitN(raw, ":", 4)

    if len(parts) < 3 {
        //return nil, errors.New("Empty message")
    }

    /*
    msgType, err := strconv.Atoi(parts[0])
    if err != nil {
        return nil, err
    }

    id := parts[1]
    endpoint := parts[2]

    data := ""
    if len(parts) == 4 {
        data = parts[3]
    }

    //return &Message{msgType, id, endpoint, data}, nil
    */
}


