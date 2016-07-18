package market

import (
    "fmt"
    "strings"
    "github.com/roydong/gmvc"
    "time"
)

type Haobtc struct {
    apiHost   string
    apiKey    string
    apiSecret string
}


func NewHaobtc() *Haobtc {
    conf := gmvc.Store.Tree("config.market.haobtc")
    hb := &Haobtc{}
    hb.apiHost, _ = conf.String("api_host")
    hb.apiKey, _ = conf.String("api_key")
    hb.apiSecret, _ = conf.String("api_secret")

    return hb
}


func (hb *Haobtc) Buy(price float64) int64 {
    p := map[string]interface{}{
        "symbol": "btc_cny",
        "type": "buy_market",
        "amount": price,
    }

    rs := hb.Call("trade", nil, p)
    if rs == nil {
        return 0
    }

    id, _ := rs.Int64("order_id")
    return id
}


func (hb *Haobtc) Sell(amount float64) int64 {
    p := map[string]interface{}{
        "symbol": "btc_cny",
        "type": "sell_market",
        "amount": amount,
    }

    rs := hb.Call("trade", nil, p)
    if rs == nil {
        return 0
    }
    id, _ := rs.Int64("order_id")
    return id
}

func (hb *Haobtc) OrderInfo(id int64) *Order {
    return &Order{}
}

func (hb *Haobtc) LastTicker() *Ticker {
    rs := hb.Call("ticker", nil, nil)
    if rs == nil {
        return nil
    }

    rst     := rs.Tree("ticker")
    t        := &Ticker{}
    t.High, _ = rst.Float("high")
    t.Low,  _ = rst.Float("low")
    t.Sell, _ = rst.Float("sell")
    t.Buy,  _ = rst.Float("buy")
    t.Last, _ = rst.Float("last")
    t.Vol,  _ = rst.Float("vol")

    //time, _ := rs.Float("date")
    //t.Time = int64(time)
    t.Time = time.Now().Unix()

    return t
}

func (hb *Haobtc) GetDepth() ([][]float64, [][]float64) {
    query := map[string]interface{}{
        "size": 50,
    }

    rs := hb.Call("depth", query, nil)
    if rs == nil {
        return nil, nil
    }

    var l int
    ask := make([][]float64, 0, l)
    l = rs.NodeNum("asks")
    for i := 0; i < l; i++ {
        price, _ := rs.Float(fmt.Sprintf("asks.%v.%v", i, 0))
        amount, _ := rs.Float(fmt.Sprintf("asks.%v.%v", i, 1))
        ask = append(ask, []float64{price, amount})
    }

    bid := make([][]float64, 0, l)
    l = rs.NodeNum("bids")
    for i := 0; i < l; i++ {
        price, _ := rs.Float(fmt.Sprintf("bids.%v.%v", i, 0))
        amount, _ := rs.Float(fmt.Sprintf("bids.%v.%v", i, 1))
        bid = append(bid, []float64{price, amount})
    }

    return ask, bid
}

func (hb *Haobtc) GetBalance() (float64, float64) {
    rs := hb.Call("account_info", nil, map[string]interface{}{})
    if rs == nil {
        return 0, 0
    }

    btc, _ := rs.Float("exchange_btc")
    cny, _ := rs.Float("exchange_cny")

    return btc, cny
}


func (hb *Haobtc) Call(api string, query, params map[string]interface{}) *gmvc.Tree {
    if params != nil {
        params["api_key"] = hb.apiKey
        params["sign"] = strings.ToUpper(createSignature(params, hb.apiSecret))
    }

    tree := CallRest(hb.apiHost + api, query, params)
    if code, has := tree.Int64("error_code"); has {
        msg, _ := tree.String("error_message")
        gmvc.Logger.Println(fmt.Sprintf("haobtc: %v %s", code, msg))
        return nil
    }

    return tree
}





