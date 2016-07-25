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
    marketHost string
    httpHost   string
    apiKey     string
    apiSecret  string
    wsHost     string

}


func NewHuobiWS() *HuobiWS {
    conf := gmvc.Store.Tree("config.market.huobi")
    hb := &HuobiWS{}
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


func (hb *HuobiWS) LastTicker() *Ticker {
    rs := hb.CallMarket("staticmarket/ticker_btc_json.js", nil, nil)
    if rs == nil {
        return nil
    }

    rst := rs.Tree("ticker")
    t := &Ticker{}
    t.Time, _ = rs.Int64("time")
    t.High, _ = rst.Float("high")
    t.Low,  _ = rst.Float("low")
    t.Sell, _ = rst.Float("sell")
    t.Buy,  _ = rst.Float("buy")
    t.Last, _ = rst.Float("last")
    t.Vol,  _ = rst.Float("vol")

    return t
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

func (hb *HuobiWS) WSConnect() {
    io, err := socketio.Dial(hb.wsHost, 3 * time.Second)
    log.Println(hb.wsHost, err)

    var strMsg = `{"symbolList":{"marketDetail":[{"symbolId":"btccny","pushType":"pushLong"}],"tradeDetail":[{"symbolId":"btccny","pushType":"pushLong"}]},"version":1,"msgType":"reqMsgSubscribe","requestIndex":103}`



    err = io.EmitText("request", strMsg)

    data := map[string]interface{}{
        "symbolIdList": []string{"btccny"},
        "version":1,
        "msgType": "reqSymbolList",
        "requestIndex": 101,
    }

    err = io.Emit("request", data)
    log.Println(err)
}


