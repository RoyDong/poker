package utils

import (
    "golang.org/x/net/websocket"
)

type WsClient struct {
    wss string
    origin string
    conn *websocket.Conn
    newMsg func(msg []byte)
    connected func()
}

func NewWsClient(wss, origin string, nm func(msg []byte), connected func()) (*WsClient, error) {
    ws := &WsClient{wss, origin, nil, nm, connected}
    err := ws.connect()
    if err != nil {
        return nil, err
    }
    go ws.readLoop()
    ws.connected()
    return ws, nil
}

func (ws *WsClient) connect() error {
    conn, err := websocket.Dial(ws.wss, "", ws.origin)
    if err != nil {
        return err
    }
    ws.conn = conn
    return nil
}

func (ws *WsClient) Close() error {
    ws.newMsg = nil
    ws.connected = nil
    return ws.conn.Close()
}

func (ws *WsClient) SendJson(msg interface{}) error {
    return websocket.JSON.Send(ws.conn, msg)
}

func (ws *WsClient) Send(msg []byte) error {
    return websocket.Message.Send(ws.conn, msg)
}

func (ws *WsClient) readLoop() {
    for {
        var msg []byte
        err := websocket.Message.Receive(ws.conn, &msg)
        if err != nil {
            err = ws.connect()
            if err != nil {
                ws.Close()
                return
            }
            ws.connected()
            continue
        }
        go ws.newMsg(msg)
    }
}

