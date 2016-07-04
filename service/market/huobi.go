package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "time"
    "fmt"
    "log"
    "github.com/gorilla/websocket"
    "net/http"
)

type Huobi struct {
    marketHost string
    httpHost    string
    apiKey     string
    apiSecret  string
    wsUrl string

}


func NewHuobi() *Huobi {
    conf := gmvc.Store.Tree("config.market.huobi")
    hb := &Huobi{}
    hb.marketHost, _ = conf.String("market_host")
    hb.httpHost, _ = conf.String("http_host")
    hb.apiKey, _ = conf.String("api_key")
    hb.apiSecret, _ = conf.String("api_secret")
    hb.wsUrl, _ = conf.String("ws_url")

    return hb
}


func (hb *Huobi) Buy(price float64) int64 {
    q := map[string]interface{}{
        "method": "buy_market",
        "coin_type": 1,
        "amount": fmt.Sprintf("%.2f", price),
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float64("id")
    return int64(id)
}


func (hb *Huobi) Sell(amount float64) int64 {
    q := map[string]interface{}{
        "method": "sell_market",
        "coin_type": 1,
        "amount": amount,
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float64("id")
    return int64(id)
}


func (hb *Huobi) OrderInfo(id int64) *Order {
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
    order.Amount,     _ = rs.Float64("order_amount")
    order.Price,      _ = rs.Float64("order_price")
    order.DealAmount, _ = rs.Float64("processed_amount")
    order.AvgPrice,   _ = rs.Float64("processed_price")

    typ, _ := rs.Float64("type")
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


func (hb *Huobi) LastTicker() *Ticker {
    rs := hb.CallMarket("staticmarket/ticker_btc_json.js", nil, nil)
    if rs == nil {
        return nil
    }

    rst := rs.Tree("ticker")
    t := &Ticker{}
    t.Time, _ = rs.Int64("time")
    t.High, _ = rst.Float64("high")
    t.Low,  _ = rst.Float64("low")
    t.Sell, _ = rst.Float64("sell")
    t.Buy,  _ = rst.Float64("buy")
    t.Last, _ = rst.Float64("last")
    t.Vol,  _ = rst.Float64("vol")

    return t
}

func (hb *Huobi) GetDepth() ([][]float64, [][]float64) {
    rs := hb.CallMarket("staticmarket/detail_btc_json.js", nil, nil)
    if rs == nil {
        return nil, nil
    }

    var l int
    ask := make([][]float64, 0, l)
    l = rs.NodeNum("sells")
    for i := 0; i < l; i++ {
        price, _ := rs.Float64(fmt.Sprintf("sells.%v.price", i))
        amount, _ := rs.Float64(fmt.Sprintf("sells.%v.amount", i))
        ask = append(ask, []float64{price, amount})
    }

    bid := make([][]float64, 0, l)
    l = rs.NodeNum("buys")
    for i := 0; i < l; i++ {
        price, _ := rs.Float64(fmt.Sprintf("buys.%v.price", i))
        amount, _ := rs.Float64(fmt.Sprintf("buys.%v.amount", i))
        bid = append(bid, []float64{price, amount})
    }

    return ask, bid
}

func (hb *Huobi) GetBalance() (float64, float64) {
    q := map[string]interface{}{
        "method": "get_account_info",
    }

    rs := hb.Call("", nil, q)
    btc, _ := rs.Float64("available_btc_display")
    cny, _ := rs.Float64("available_cny_display")

    return btc,cny
}

func (hb *Huobi) Call(api string, query, params map[string]interface{}) *gmvc.Tree {
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

func (hb *Huobi) CallMarket(api string, query, params map[string]interface{}) *gmvc.Tree {
    tree := CallRest(hb.marketHost + api, query, params)
    if code, has := tree.Int64("code"); has {
        msg, _ := tree.String("message")
        gmvc.Logger.Println(fmt.Sprintf("huobi: %v %s", code, msg))
        return nil
    }
    return tree
}

func (hb *Huobi) WSConnect() {

    dialer := &websocket.Dialer{
        HandshakeTimeout: 3 * time.Second,
    }

    conn, _, err := dialer.Dial("wss://hq.huobi.com:80", http.Header{})
    log.Println(conn,err)

    /*
    data := map[string]interface{}{
        "symbolIdList": []string{"btccny"},
        "version": 1,
        "msgType": "reqSymbolList",
        "requestIndex": 100,
    }

    err = conn.WriteJSON(data)
    log.Println(err)


    var rs interface{}
    err = conn.ReadJSON(&rs)

    log.Println(rs, err)
    */
}



