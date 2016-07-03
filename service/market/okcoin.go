package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "fmt"
)

type OKCoin struct {
    httpHost   string
    apiKey    string
    apiSecret string

    wsUrl string
}


func NewOKCoin() *OKCoin {
    conf := gmvc.Store.Tree("config.market.okcoin")
    ok := &OKCoin{}
    ok.httpHost, _ = conf.String("http_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")
    ok.wsUrl, _ = conf.String("ws_url")

    return ok
}


func (ok *OKCoin)Buy(price float64) int64 {
    p := map[string]interface{}{
        "symbol": "btc_cny",
        "type": "buy_market",
        "price": price,
    }

    rs := ok.Call("trade.do", nil, p)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float64("order_id")
    return int64(id)
}


func (ok *OKCoin)Sell(amount float64) int64 {
    p := map[string]interface{}{
        "symbol": "btc_cny",
        "type": "sell_market",
        "amount": amount,
    }

    rs := ok.Call("trade.do", nil, p)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float64("order_id")
    return int64(id)
}

func (ok *OKCoin) OrderInfo(id int64) *Order {
    params := map[string]interface{}{
        "symbol": "btc_cny",
        "order_id": id,
    }

    rs := ok.Call("order_info.do", nil, params)
    if rs == nil {
        return nil
    }

    rst := rs.Tree("orders.0")
    if rst == nil {
        return nil
    }

    order := &Order{}
    order.Id = id

    order.Amount, _ = rst.Float64("amount")
    order.Price, _ = rst.Float64("price")
    order.DealAmount, _ = rst.Float64("deal_amount")
    order.AvgPrice, _ = rst.Float64("avg_price")

    t, _ := rst.Float64("create_date")
    order.Created = int64(t)

    return order
}


func (ok *OKCoin) LastTicker() *Ticker {
    rs := ok.Call("ticker.do", map[string]interface{}{"symbol": "btc_cny"}, nil)
    if rs == nil {
        return nil
    }

    rst     := rs.Tree("ticker")
    t         := &Ticker{}
    t.Time, _ = rs.Int64("date")
    t.High, _ = rst.Float64("high")
    t.Low,  _ = rst.Float64("low")
    t.Sell, _ = rst.Float64("sell")
    t.Buy,  _ = rst.Float64("buy")
    t.Last, _ = rst.Float64("last")
    t.Vol,  _ = rst.Float64("vol")

    return t
}

func (ok *OKCoin) GetDepth() ([][]float64, [][]float64) {
    query := map[string]interface{}{
        "symbol": "btc_cny",
        "size": 50,
        "merge": 0,
    }

    rs := ok.Call("depth.do", query, nil)
    if rs == nil {
        return nil, nil
    }

    var l int
    ask := make([][]float64, 0, l)
    l = rs.NodeNum("asks")
    for i := l - 1; i >= 0; i-- {
        price, _ := rs.Float64(fmt.Sprintf("asks.%v.%v", i, 0))
        amount, _ := rs.Float64(fmt.Sprintf("asks.%v.%v", i, 1))
        ask = append(ask, []float64{price, amount})
    }

    bid := make([][]float64, 0, l)
    l = rs.NodeNum("bids")
    for i := 0; i < l; i++ {
        price, _ := rs.Float64(fmt.Sprintf("bids.%v.%v", i, 0))
        amount, _ := rs.Float64(fmt.Sprintf("bids.%v.%v", i, 1))
        bid = append(bid, []float64{price, amount})
    }

    return ask, bid
}

func (ok *OKCoin) GetBalance() (float64, float64) {
    rs := ok.Call("userinfo.do", nil, map[string]interface{}{})
    free := rs.Tree("info.funds.free")
    if free == nil {
        return 0, 0
    }

    btc, _ := free.Float64("btc")
    cny, _ := free.Float64("cny")

    return btc,cny
}


func (ok *OKCoin) Call(api string, query, params map[string]interface{}) *gmvc.Tree {
    if params != nil {
        params["api_key"] = ok.apiKey
        params["sign"] = strings.ToUpper(createSignature(params, ok.apiSecret))
    }

    tree := CallRest(ok.httpHost + api, query, params)
    if code, has := tree.Int64("error_code"); has {
        gmvc.Logger.Println(fmt.Sprintf("okcoin: %v %s", code))
        return nil
    }
    return tree
}






