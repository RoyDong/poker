package market

import (
    "github.com/roydong/gmvc"
    "fmt"
    "time"
    "github.com/gorilla/websocket"
    "strings"
    "log"
)


type OKFutureWS struct {
    *gmvc.Event

    wsHost   string
    apiKey    string
    apiSecret string

    contractType string

    conn *websocket.Conn

    lastTicker *Ticker
    lastTickerUnix int64

    lastAsks [][]float64
    lastBids [][]float64
    lastDepthUnix int64

    btc float64
    lastBalanceUnix int64

    lastMsgUnix int64

    addedChannels []string
}

func NewOKFutureWS(contractType string) *OKFutureWS {
    ok := &OKFutureWS{}

    conf := gmvc.Store.Tree("config.market.okfuture")
    ok.Event = gmvc.NewEvent()
    ok.wsHost, _ = conf.String("ws_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")
    ok.contractType = contractType
    ok.addedChannels = make([]string, 0, 5)

    ok.Connect()
    go ok.readLoop()

    ok.AddChannels()

    return ok
}

func (ok *OKFutureWS) syncTicker(args ...interface{}) {
    rs, _ := args[0].(*gmvc.Tree)

    t := &Ticker{}
    t.High, _ = rs.Float64("high")
    t.Low,  _ = rs.Float64("low")
    t.Sell, _ = rs.Float64("sell")
    t.Buy,  _ = rs.Float64("buy")
    t.Last, _ = rs.Float64("last")
    t.Vol,  _ = rs.Float64("volume")
    t.Time    = time.Now().Unix()
    ok.lastTickerUnix = t.Time

    log.Println(t)
    ok.lastTicker = t
}

func (ok *OKFutureWS) LastTicker() *Ticker {
    return ok.lastTicker
}

func (ok *OKFutureWS) syncDepth(args ...interface{}) {
    rs, _ := args[0].(*gmvc.Tree)

    var l int
    asks := make([][]float64, 0, l)
    l = rs.NodeNum("asks")
    for i := l - 1; i >= 0; i-- {
        price, _ := rs.Float64(fmt.Sprintf("asks.%v.0", i))
        amount, _ := rs.Float64(fmt.Sprintf("asks.%v.1", i))
        asks = append(asks, []float64{price, amount})
    }

    bids := make([][]float64, 0, l)
    l = rs.NodeNum("bids")
    for i := 0; i < l; i++ {
        price, _ := rs.Float64(fmt.Sprintf("bids.%v.0", i))
        amount, _ := rs.Float64(fmt.Sprintf("bids.%v.1", i))
        bids = append(bids, []float64{price, amount})
    }

    ok.lastAsks = asks
    ok.lastBids = bids
    ok.lastDepthUnix = time.Now().Unix()
}

func (ok *OKFutureWS) GetDepth() ([][]float64, [][]float64) {
    return ok.lastAsks, ok.lastBids
}

func (ok *OKFutureWS) syncBalance(args ...interface{}) {
    rs, _ := args[0].(*gmvc.Tree)

    ok.btc, _ = rs.Float64("info.btc.balance")
    ok.lastBalanceUnix = time.Now().Unix()
    log.Println(ok.btc)
}

func (ok *OKFutureWS) GetBalance() (float64, float64) {
    return ok.btc, 0
}

func (ok *OKFutureWS) Connect() {
    var err error
    ok.conn, _, err = (&websocket.Dialer{}).Dial(ok.wsHost, nil)
    if err != nil {
        gmvc.Logger.Fatalln(err)
    }
}

func (ok *OKFutureWS) Stop() {
    ok.RemoveChannels()
    ok.conn.Close()
}

func (ok *OKFutureWS) AddChannel(name string, params map[string]interface{}) {
    msg := map[string]interface{} {
        "event": "addChannel",
        "channel": name,
    }
    if len(params) > 0 {
        msg["parameters"] = params
    }
    err := ok.conn.WriteJSON(msg)
    if err != nil {
        gmvc.Logger.Fatalln("okfuture add channel failed")
    }
}

func (ok *OKFutureWS) signParams(params map[string]interface{}) map[string]interface{} {
    if params == nil {
        params = make(map[string]interface{}, 2)
    }
    params["api_key"] = ok.apiKey
    params["sign"] = strings.ToUpper(createSignature(params, ok.apiSecret))
    return params
}

func (ok *OKFutureWS) AddChannels() {
    channel := "ok_sub_futureusd_btc_ticker_" + ok.contractType
    ok.AddChannel(channel, nil)
    ok.AddHandler(channel, ok.syncTicker)
    ok.addedChannels = append(ok.addedChannels, channel)

    channel = fmt.Sprintf("ok_sub_futureusd_btc_depth_%s_%d", ok.contractType, 20)
    ok.AddChannel(channel, nil)
    ok.AddHandler(channel, ok.syncDepth)
    ok.addedChannels = append(ok.addedChannels, channel)

    channel = "ok_futureusd_userinfo"
    ok.AddChannel(channel, ok.signParams(nil))
    ok.AddHandler(channel, ok.syncBalance)
    ok.addedChannels = append(ok.addedChannels, channel)
}

func (ok *OKFutureWS) RemoveChannel(name string) {
    err := ok.conn.WriteMessage(websocket.TextMessage,
        []byte(fmt.Sprintf(`{"event":"removeChannel","channel":"%s"}`, name)))
    if err != nil {
        gmvc.Logger.Fatalln("okfuture remove channel failed " + name)
    }
}

func (ok *OKFutureWS) RemoveChannels() {
    query := make([]string, 0, len(ok.addedChannels))
    for _, channel := range ok.addedChannels {
        query = append(query, fmt.Sprintf(`{"event":"removeChannel","channel":"%s"}`, channel))
    }
    err := ok.conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("[%s]", strings.Join(query, ","))))
    if err != nil {
        gmvc.Logger.Fatalln("okfuture remove channels failed")
    }
}

func (ok *OKFutureWS) Ping() {
    ok.conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"ping"}`))
}

func (ok *OKFutureWS) readLoop() {
   for {
       typ, raw, err := ok.conn.ReadMessage()
       if err != nil {
           gmvc.Logger.Println("okfuture ws read message error ", err.Error())
           return
       }
       if typ != websocket.TextMessage {
           gmvc.Logger.Println("okfuture ws not text message", err.Error())
           continue
       }
       rs := gmvc.NewTree()
       err = rs.LoadJson("", raw, true)
       if err != nil {
           continue
       }
       ok.lastMsgUnix = time.Now().Unix()
       for i, l := 0, rs.NodeNum(""); i < l; i++ {
           event := rs.Tree(fmt.Sprintf("%d", i))
           if event == nil {
               gmvc.Logger.Println("okfuture ws error " + string(raw))
               continue
           }
           if _, has := event.Float64("errorcode"); has {
               gmvc.Logger.Println("okfuture ws error " + string(raw))
               continue
           }
           if name, has := event.String("channel"); has && len(name) > 0 {
               if data := event.Tree("data"); data != nil {
                   ok.Trigger(name, data)
               }
           }
       }
   }
}



