package market

import (
    "github.com/roydong/gmvc"
    "fmt"
    "time"
    "github.com/gorilla/websocket"
    "strings"
    "sync"
)

const (
    TypeOpenLong = 1
    TypeOpenShort = 2
    TypeCloseLong = 3
    TypeCloseShort = 4
)

type OKFutureWS struct {
    *gmvc.Event

    wsHost   string
    apiKey    string
    apiSecret string
    contractType string
    leverRate int

    conn *websocket.Conn

    lastTicker Ticker
    lastAsks [][]float64
    lastBids [][]float64

    lastBtc chan float64
    lastBtcLocker *sync.Mutex

    lastOrderId chan int64
    lastOrderIdLocker *sync.Mutex

    cancelOrderId chan int64
    cancelOrderIdLocker *sync.Mutex

    tickerUpdated int64
    depthUpdated int64
    balanceUpdated int64
    msgUpdated int64

    currentOrders map[int64]Order

    subChannels []string
}

func NewOKFutureWS(contractType string) *OKFutureWS {
    conf := gmvc.Store.Tree("config.market.okfuture")
    ok := &OKFutureWS{}
    ok.Event = gmvc.NewEvent()
    ok.wsHost, _ = conf.String("ws_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")
    ok.contractType = contractType
    ok.leverRate = 10

    ok.lastTicker = Ticker{}

    ok.lastBtc = make(chan float64)
    ok.lastBtcLocker = &sync.Mutex{}

    ok.lastOrderId = make(chan int64)
    ok.lastOrderIdLocker = &sync.Mutex{}

    ok.cancelOrderId = make(chan int64)
    ok.cancelOrderIdLocker = &sync.Mutex{}

    ok.currentOrders = make(map[int64]Order, 4)

    ok.subChannels = []string{
        "ok_sub_futureusd_btc_ticker_" + ok.contractType,
        fmt.Sprintf("ok_sub_futureusd_btc_depth_%s_%d", ok.contractType, 20),
        "ok_sub_futureusd_trades",
    }

    ok.initCallbacks()
    ok.Connect()

    return ok
}

func (ok *OKFutureWS) syncTicker(args ...interface{}) {
    rs, _ := args[0].(*gmvc.Tree)
    if rs == nil {
        return
    }

    t := Ticker{}
    t.High, _ = rs.Float("high")
    t.Low,  _ = rs.Float("low")
    t.Sell, _ = rs.Float("sell")
    t.Buy,  _ = rs.Float("buy")
    t.Last, _ = rs.Float("last")
    t.Vol,  _ = rs.Float("volume")
    t.Time    = time.Now().Unix()
    ok.tickerUpdated = t.Time
    ok.lastTicker = t
}

func (ok *OKFutureWS) Name() string {
    return ok.contractType
}

func (ok *OKFutureWS) LastTicker() Ticker {
    return ok.lastTicker
}

func (ok *OKFutureWS) syncDepth(args ...interface{}) {
    rs, _ := args[0].(*gmvc.Tree)
    if rs == nil {
        gmvc.Logger.Println("depth data is nil")
        return
    }

    l := rs.NodeNum("asks")
    asks := make([][]float64, 0, l)
    for i := l - 1; i >= 0; i-- {
        price, _ := rs.Float(fmt.Sprintf("asks.%v.0", i))
        amount, _ := rs.Float(fmt.Sprintf("asks.%v.1", i))
        asks = append(asks, []float64{price, amount})
    }

    l = rs.NodeNum("bids")
    bids := make([][]float64, 0, l)
    for i := 0; i < l; i++ {
        price, _ := rs.Float(fmt.Sprintf("bids.%v.0", i))
        amount, _ := rs.Float(fmt.Sprintf("bids.%v.1", i))
        bids = append(bids, []float64{price, amount})
    }

    ok.lastAsks = asks
    ok.lastBids = bids
    ok.depthUpdated = time.Now().Unix()
    ok.Trigger("depth", asks, bids, ok.depthUpdated)
}

func (ok *OKFutureWS) GetDepth() ([][]float64, [][]float64) {
    return ok.lastAsks, ok.lastBids
}

func (ok *OKFutureWS) DepthUpdated() int64 {
    return ok.depthUpdated
}

func (ok *OKFutureWS) syncBalance(args ...interface{}) {
    code, _ := args[1].(int64)
    var btc float64
    if code > 0 {
        btc = 0
    } else {
        rs, _ := args[0].(*gmvc.Tree)
        btc, _ = rs.Float("info.btc.balance")
    }
    ok.balanceUpdated = time.Now().Unix()
    ok.lastBtc <-btc
}

func (ok *OKFutureWS) GetBalance() (float64, float64) {
    ok.lastBtcLocker.Lock()
    defer ok.lastBtcLocker.Unlock()
    ok.addChannel("ok_futureusd_userinfo", make(map[string]interface{}))
    return <-ok.lastBtc, 0
}

func (ok *OKFutureWS) OpenPosition(typ int, amount int64, price float64) int64 {
    ok.lastOrderIdLocker.Lock()
    defer ok.lastOrderIdLocker.Unlock()
    params := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": ok.contractType,
        "type": typ,
        "amount": amount,
        "price": price,
        "match_price": 0,
        "lever_rate": ok.leverRate,
    }
    ok.addChannel("ok_futuresusd_trade", params)
    return <-ok.lastOrderId
}

func (ok *OKFutureWS) syncOpenResult(args ...interface{}) {
    var id int64
    rs, _ := args[0].(*gmvc.Tree)
    if rs != nil {
        id, _ = rs.Int64("order_id")
    }
    ok.lastOrderId <-id
}

func (ok *OKFutureWS) CancelOrder(id int64) int64 {
    ok.cancelOrderIdLocker.Lock()
    defer ok.cancelOrderIdLocker.Unlock()
    params := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": ok.contractType,
        "order_id": id,
    }
    ok.addChannel("ok_futureusd_cancel_order", params)
    return <-ok.cancelOrderId
}

func (ok *OKFutureWS) syncCancelResult(args ...interface{}) {
    var id int64
    rs, _ := args[0].(*gmvc.Tree)
    if rs != nil {
        id, _ = rs.Int64("order_id")
    }
    ok.cancelOrderId <-id
}

func (ok *OKFutureWS) syncCurrentOrder(args ...interface{}) {
    rs, _ := args[0].(*gmvc.Tree)
    if rs == nil {
        return
    }

    order := Order{}
    order.Id, _ = rs.Int64("orderid")
    order.Amount, _ = rs.Float("amount")
    order.Price, _ = rs.Float("price")
    order.DealAmount, _ = rs.Float("deal_amount")
    order.AvgPrice, _ = rs.Float("price_avg")
    order.Status, _ = rs.Int("status")
    order.Fee, _ = rs.Float("fee")

    ok.currentOrders[order.Id] = order
    ok.Trigger("order", order)
}

func (ok *OKFutureWS) Connect() {
    var err error
    ok.conn, _, err = (&websocket.Dialer{}).Dial(ok.wsHost, nil)
    if err != nil {
        gmvc.Logger.Fatalln(err)
    }

    go ok.readLoop()
    ok.addChannel(ok.subChannels[0], nil)
    ok.addChannel(ok.subChannels[1], nil)
    ok.addChannel(ok.subChannels[2], map[string]interface{}{})
}

func (ok *OKFutureWS) Stop() {
    ok.removeChannels()
    ok.conn.Close()
}

func (ok *OKFutureWS) signParams(params map[string]interface{}) map[string]interface{} {
    if params == nil {
        params = make(map[string]interface{}, 2)
    }
    params["api_key"] = ok.apiKey
    params["sign"] = strings.ToUpper(createSignature(params, ok.apiSecret))
    return params
}

func (ok *OKFutureWS) addChannel(name string, params map[string]interface{}) {
    msg := map[string]interface{} {
        "event": "addChannel",
        "channel": name,
    }
    if params != nil {
        msg["parameters"] = ok.signParams(params)
    }
    err := ok.conn.WriteJSON(msg)
    if err != nil {
        gmvc.Logger.Fatalln("okfuture add channel failed")
    }
}

func (ok *OKFutureWS) initCallbacks() {
    ok.AddHandler(ok.subChannels[0], ok.syncTicker)
    ok.AddHandler(ok.subChannels[1], ok.syncDepth)
    ok.AddHandler(ok.subChannels[2], ok.syncCurrentOrder)
    ok.AddHandler("ok_futureusd_userinfo", ok.syncBalance)
    ok.AddHandler("ok_futuresusd_trade", ok.syncOpenResult)
    ok.AddHandler("ok_futureusd_cancel_order", ok.syncCancelResult)
}

func (ok *OKFutureWS) RemoveChannel(name string) {
    err := ok.conn.WriteMessage(websocket.TextMessage,
        []byte(fmt.Sprintf(`{"event":"removeChannel","channel":"%s"}`, name)))
    if err != nil {
        gmvc.Logger.Fatalln("okfuture remove channel failed " + name)
    }
}

func (ok *OKFutureWS) removeChannels() {
    query := make([]string, 0, len(ok.subChannels))
    for _, channel := range ok.subChannels{
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
       ok.msgUpdated = time.Now().Unix()
       for i, l := 0, rs.NodeNum(""); i < l; i++ {
           event := rs.Tree(fmt.Sprintf("%d", i))
           if event == nil {
               gmvc.Logger.Println("okfuture ws error " + string(raw))
               continue
           }
           if name, has := event.String("channel"); has && len(name) > 0 {
               data := event.Tree("data")
               code, _ := event.Int64("errorcode")
               if code > 0 {
                   gmvc.Logger.Println("okfuture ws error " + string(raw))
               }
               ok.Trigger(name, data, code)
           }
       }
   }
}



