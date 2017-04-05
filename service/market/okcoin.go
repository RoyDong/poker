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

func (ok *OKCoin) OpenLong(price, amount float64) int64 {
    return ok.Buy(price, amount)
}

func (ok *OKCoin) CloseLong(price, amount float64) int64 {
    return ok.Sell(price, amount)
}

func (ok *OKCoin) OpenShort(price, amount float64) int64 {
    return ok.Sell(price, amount)
}

func (ok *OKCoin) CloseShort(price, amount float64) int64 {
    return ok.Buy(price, amount)
}

func (ok *OKCoin) Buy(price, amount float64) int64 {
    p := map[string]interface{}{
        "symbol": "btc_cny",
        "price": price,
    }

    if amount > 0 {
        p["type"] = "buy"
        p["amount"] = amount
    } else {
        p["type"] = "buy_market"
    }

    rs := ok.Call("trade.do", nil, p)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float("order_id")
    return int64(id)
}


func (ok *OKCoin) Sell(price, amount float64) int64 {
    p := map[string]interface{}{
        "symbol": "btc_cny",
        "type": "sell_market",
        "amount": amount,
    }

    if price > 0 {
        p["type"] = "sell"
        p["price"] = price
    } else {
        p["type"] = "sell_market"
    }

    rs := ok.Call("trade.do", nil, p)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float("order_id")
    return int64(id)
}

func (ok *OKCoin) OrderInfo(id int64) Order {
    params := map[string]interface{}{
        "symbol": "btc_cny",
        "order_id": id,
    }

    order := Order{}
    rs := ok.Call("order_info.do", nil, params)
    if rs == nil {
        return order
    }

    rst := rs.Tree("orders.0")
    if rst == nil {
        return order
    }

    order.Id = id

    order.Amount, _ = rst.Float("amount")
    order.Price, _ = rst.Float("price")
    order.DealAmount, _ = rst.Float("deal_amount")
    order.AvgPrice, _ = rst.Float("avg_price")

    order.Status, _ = rst.Int("status")

    t, _ := rst.Float("create_date")
    order.Created = int64(t)

    return order
}

func (ok *OKCoin) CancelOrder(id int64) bool {
    params := map[string]interface{} {
        "symbol": "btc_cny",
        "order_id": id,
    }

    rs := ok.Call("cancel_order.do", nil, params)
    if rs == nil {
        return false
    }

    if result, _ := rs.String("result"); result == "true" {
        return true
    }

    return false
}

func (ok *OKCoin) LastTicker() Ticker {
    t := Ticker{}
    rs := ok.Call("ticker.do", map[string]interface{}{"symbol": "btc_cny"}, nil)
    if rs == nil {
        return t
    }

    rst     := rs.Tree("ticker")
    t.Time, _ = rs.Int64("date")
    t.High, _ = rst.Float("high")
    t.Low,  _ = rst.Float("low")
    t.Sell, _ = rst.Float("sell")
    t.Buy,  _ = rst.Float("buy")
    t.Last, _ = rst.Float("last")
    t.Vol,  _ = rst.Float("vol")

    return t
}

func (ok *OKCoin) GetDepth() ([][]float64, [][]float64) {
    query := map[string]interface{}{
        "symbol": "btc_cny",
        "size": 50,
    }

    rs := ok.Call("depth.do", query, nil)
    if rs == nil {
        return nil, nil
    }

    l := rs.NodeNum("asks")
    ask := make([][]float64, 0, l)
    for i := l - 1; i >= 0; i-- {
        rst := rs.Tree(fmt.Sprintf("asks.%v", i))
        price, _ := rst.Float("0")
        amount, _ := rst.Float("1")
        ask = append(ask, []float64{price, amount})
    }

    l = rs.NodeNum("bids")
    bid := make([][]float64, 0, l)
    for i := 0; i < l; i++ {
        rst := rs.Tree(fmt.Sprintf("bids.%v", i))
        price, _ := rst.Float("0")
        amount, _ := rst.Float("1")
        bid = append(bid, []float64{price, amount})
    }

    return ask, bid
}

func (ok *OKCoin) GetBalance() (float64, float64) {
    rs := ok.Call("userinfo.do", nil, map[string]interface{}{})
    if rs == nil {
        return 0, 0
    }

    free := rs.Tree("info.funds.free")
    btc, _ := free.Float("btc")
    cny, _ := free.Float("cny")

    return btc,cny
}


func (ok *OKCoin) Call(api string, query, params map[string]interface{}) *gmvc.Tree {
    if params != nil {
        params["api_key"] = ok.apiKey
        params["sign"] = strings.ToUpper(createSignature(params, ok.apiSecret))
    }

    tree := CallRest(ok.httpHost + api, query, params)
    if tree == nil {
        return nil
    }

    if code, has := tree.Int64("error_code"); has {
        gmvc.Logger.Println(fmt.Sprintf("okcoin: %v %s", code))
        return nil
    }
    return tree
}






