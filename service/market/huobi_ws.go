package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "time"
    "fmt"
    "log"
    "github.com/roydong/poker/lib/socketio"
)

type HuobiWS struct {
    *gmvc.Event

    marketHost string
    httpHost   string
    apiKey     string
    apiSecret  string
    wsHost     string

    socketio   *socketio.Socket

    lastAsks   [][]float64
    lastBids   [][]float64
}


func NewHuobiWS() *HuobiWS {
    hb := &HuobiWS{}

    hb.Event = &gmvc.Event{}

    conf := gmvc.Store.Tree("config.market.huobi")
    hb.marketHost, _ = conf.String("market_host")
    hb.httpHost, _ = conf.String("http_host")
    hb.apiKey, _ = conf.String("api_key")
    hb.apiSecret, _ = conf.String("api_secret")
    hb.wsHost, _ = conf.String("ws_host")

    return hb
}


func (hb *HuobiWS) Buy(price float64) int64 {
    q := map[string]interface{}{
        "method": "buy_market",
        "coin_type": 1,
        "amount": fmt.Sprintf("%.2f", price),
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float("id")
    return int64(id)
}


func (hb *HuobiWS) Sell(amount float64) int64 {
    q := map[string]interface{}{
        "method": "sell_market",
        "coin_type": 1,
        "amount": amount,
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float("id")
    return int64(id)
}


func (hb *HuobiWS) OrderInfo(id int64) *Order {
    params := map[string]interface{}{
        "method": "order_info",
        "coin_type": 1,
        "id": id,
    }

    rs := hb.Call("", nil, params)
    if rs == nil {
        return nil
    }

    order := &Order{}
    order.Id = id
    order.Amount,     _ = rs.Float("order_amount")
    order.Price,      _ = rs.Float("order_price")
    order.DealAmount, _ = rs.Float("processed_amount")
    order.AvgPrice,   _ = rs.Float("processed_price")

    typ, _ := rs.Float("type")
    if int64(typ) == 3 {
        order.Price = order.Amount
        order.Amount = 0
        if order.DealAmount > 0 && order.AvgPrice > 0 {
            order.DealAmount = order.DealAmount / order.AvgPrice
        }
    }

    order.Created = time.Now().Unix()

    return order
}


func (hb *HuobiWS) GetDepth() ([][]float64, [][]float64) {
    rs := hb.CallMarket("staticmarket/detail_btc_json.js", nil, nil)
    if rs == nil {
        return nil, nil
    }

    var l int
    ask := make([][]float64, 0, l)
    l = rs.NodeNum("sells")
    for i := 0; i < l; i++ {
        price, _ := rs.Float(fmt.Sprintf("sells.%v.price", i))
        amount, _ := rs.Float(fmt.Sprintf("sells.%v.amount", i))
        ask = append(ask, []float64{price, amount})
    }

    bid := make([][]float64, 0, l)
    l = rs.NodeNum("buys")
    for i := 0; i < l; i++ {
        price, _ := rs.Float(fmt.Sprintf("buys.%v.price", i))
        amount, _ := rs.Float(fmt.Sprintf("buys.%v.amount", i))
        bid = append(bid, []float64{price, amount})
    }

    return ask, bid
}


func (hb *HuobiWS) GetBalance() (float64, float64) {
    q := map[string]interface{}{
        "method": "get_account_info",
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return 0, 0
    }

    btc, _ := rs.Float("available_btc_display")
    cny, _ := rs.Float("available_cny_display")

    return btc,cny
}

func (hb *HuobiWS) Call(api string, query, params map[string]interface{}) *gmvc.Tree {
    if params != nil {
        params["access_key"] = hb.apiKey
        params["created"] = time.Now().Unix()
        params["sign"] = strings.ToLower(createSignature(params, hb.apiSecret))
    }

    tree := CallRest(hb.httpHost + api, query, params)
    if code, has := tree.Int64("code"); has {
        msg, _ := tree.String("msg")
        gmvc.Logger.Println(fmt.Sprintf("huobi: %v %s", code, msg))
        return nil
    }

    return tree
}

func (hb *HuobiWS) CallMarket(api string, query, params map[string]interface{}) *gmvc.Tree {
    tree := CallRest(hb.marketHost + api, query, params)
    if code, has := tree.Int64("code"); has {
        msg, _ := tree.String("message")
        gmvc.Logger.Println(fmt.Sprintf("huobi: %v %s", code, msg))
        return nil
    }
    return tree
}

func (hb *HuobiWS) Connect() {
    io, err := socketio.Dial(hb.wsHost, 3 * time.Second)
    if err != nil {
        gmvc.Logger.Fatalln(err)
    }

    hb.socketio = io
    go hb.readLoop()
    hb.addChannels()
}

func (hb *HuobiWS) addChannels() {
    var strMsg = `{"symbolList":{
                        "marketDetail":[{"symbolId":"btccny","pushType":"pushLong"}],
                        "tradeDetail":[{"symbolId":"btccny","pushType":"pushLong"}]
                        },"version":1,"msgType":"reqMsgSubscribe","requestIndex":103}`
    err := hb.socketio.EmitText("request", strMsg)
    if err != nil {
        gmvc.Logger.Println("huobi ws error")
    }
}


func (hb *HuobiWS) readLoop() {
    for {
        msg, err := hb.socketio.Read()
        if err != nil {
            return
        }
        log.Println(1, msg.Type, string(msg.Data))
    }
}

func (hb *HuobiWS) syncTrade() {

}


