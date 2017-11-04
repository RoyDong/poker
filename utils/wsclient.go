package utils

import (
    "github.com/gorilla/websocket"
    "time"
)

type WsClient struct {
    wss string
    conn *websocket.Conn
    newMsg func(msg []byte)
    connected func()

    reconnectNum int
    reconnectGap time.Duration

    pingGap time.Duration

    msgPipe chan []byte
    jsonPipe chan interface{}
    pingPipe chan []byte
    pongPipe chan []byte
    inLoop bool
}

func NewWsClient(wss string, nm func(msg []byte), connected func()) *WsClient {
    ws := &WsClient{}
    ws.wss = wss
    ws.newMsg = nm
    ws.connected = connected

    ws.reconnectNum = -1
    ws.reconnectGap = 200 * time.Millisecond
    ws.pingGap = 2 * time.Minute

    ws.msgPipe = make(chan []byte, 5)
    ws.jsonPipe = make(chan interface{}, 5)
    ws.pingPipe = make(chan []byte, 1)
    ws.pongPipe = make(chan []byte, 1)
    return ws
}

func (ws *WsClient) Start() error {
    err := ws.connect()
    if err != nil {
        return err
    }
    ws.inLoop = true
    go ws.readLoop()
    go ws.writeLoop()
    return nil
}

func (ws *WsClient) connect() error {
    conn, _, err := (&websocket.Dialer{}).Dial(ws.wss, nil)
    if err != nil {
        return err
    }
    conn.SetPingHandler(func(appData string) error {
        ws.pongPipe <-[]byte(appData)
        return nil
    })
    conn.SetPongHandler(func(appData string) error {
        //TODO
        return nil
    })
    ws.conn = conn
    ws.connected()
    return nil
}

func (ws *WsClient) Close() error {
    ws.inLoop = false
    ws.newMsg = nil
    ws.connected = nil
    return ws.conn.Close()
}

func (ws *WsClient) SendJson(d interface{}) {
    ws.jsonPipe <-d
}

func (ws *WsClient) Send(msg []byte) {
    ws.msgPipe <-msg
}

func (ws *WsClient) writeLoop() {
    for ws.inLoop {
        var err error
        select {
        case msg := <-ws.msgPipe:
            err = ws.conn.WriteMessage(websocket.TextMessage, msg)
        case msg := <-ws.jsonPipe:
            err = ws.conn.WriteJSON(msg)
        case msg := <-ws.pingPipe:
            err = ws.conn.WriteMessage(websocket.PingMessage, msg)
        case msg := <-ws.pongPipe:
            err = ws.conn.WriteMessage(websocket.PongMessage, msg)
        }
        if err != nil {
            WarningLog.Write(err.Error())
        }
    }
}

func (ws *WsClient) readLoop() {
    pingTime := time.Now()
    reconnectNum := 0
    for ws.inLoop {
        mtype, msg, err := ws.conn.ReadMessage()
        if err != nil {
            for {
                if ws.reconnectNum >= 0 {
                    reconnectNum++
                    if reconnectNum > ws.reconnectNum {
                        ws.Close()
                        return
                    }
                }
                if ws.connect() == nil {
                    break
                }
                time.Sleep(ws.reconnectGap)
            }
            pingTime = time.Now()
            reconnectNum = 0
            continue
        }

        switch mtype {
        case websocket.TextMessage:
            go ws.newMsg(msg)
        case websocket.BinaryMessage:
            //TODO
        }

        if ws.pingGap > 0 && time.Now().Sub(pingTime) >= ws.pingGap {
            ws.pingPipe <-[]byte("p")
            pingTime = time.Now()
        }
    }
}

