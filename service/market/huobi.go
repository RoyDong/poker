package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "time"
    "fmt"
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


func (hb *Huobi) OpenLong(price, amount float64) int64 {
    return hb.Buy(price, amount)
}

func (hb *Huobi) CloseLong(price, amount float64) int64 {
    return hb.Sell(price, amount)
}

func (hb *Huobi) OpenShort(price, amount float64) int64 {
    return hb.Sell(price, amount)
}

func (hb *Huobi) CloseShort(price, amount float64) int64 {
    return hb.Buy(price, amount)
}

func (hb *Huobi) Buy(price, amount float64) int64 {
    q := make(map[string]interface{}, 4)
    q["coin_type"] = 1
    if amount > 0 {
        q["method"] = "buy"
        q["price"] = fmt.Sprintf("%.2f", price)
        q["amount"] = fmt.Sprintf("%.4f", amount)
    } else {
        q["method"] = "buy_market"
        q["amount"] = fmt.Sprintf("%.2f", price)
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float("id")
    return int64(id)
}


func (hb *Huobi) Sell(price, amount float64) int64 {
    q := make(map[string]interface{}, 4)
    q["coin_type"] = 1
    if price > 0 {
        q["method"] = "sell"
        q["price"] = fmt.Sprintf("%.2f", price)
        q["amount"] = fmt.Sprintf("%.4f", amount)
    } else {
        q["method"] = "buy_market"
        q["amount"] = fmt.Sprintf("%.4f", amount)
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return 0
    }
    id, _ := rs.Float("id")
    return int64(id)
}


func (hb *Huobi) OrderInfo(id int64) Order {
    params := map[string]interface{}{
        "method": "order_info",
        "coin_type": 1,
        "id": id,
    }

    order := Order{}
    rs := hb.Call("", nil, params)
    if rs == nil {
        return order
    }

    order.Id = id
    order.Amount,     _ = rs.Float("order_amount")
    order.Price,      _ = rs.Float("order_price")
    order.DealAmount, _ = rs.Float("processed_amount")
    order.AvgPrice,   _ = rs.Float("processed_price")
    order.Status,     _ = rs.Int("status")

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

func (hb *Huobi) CancelOrder(id int64) bool {
    params := map[string]interface{} {
        "method": "cancel_order",
        "coin_type": 1,
        "id": id,
    }

    rs := hb.Call("", nil, params)
    if rs == nil {
        return false
    }

    if result, _ := rs.String("result"); result == "success" {
        return true
    }

    return false
}

func (hb *Huobi) LastTicker() Ticker {
    t := Ticker{}
    rs := hb.CallMarket("staticmarket/ticker_btc_json.js", nil, nil)
    if rs == nil {
        return t
    }

    rst := rs.Tree("ticker")
    t.Time, _ = rs.Int64("time")
    t.High, _ = rst.Float("high")
    t.Low,  _ = rst.Float("low")
    t.Sell, _ = rst.Float("sell")
    t.Buy,  _ = rst.Float("buy")
    t.Last, _ = rst.Float("last")
    t.Vol,  _ = rst.Float("vol")

    return t
}

func (hb *Huobi) GetDepth() ([][]float64, [][]float64) {
    rs := hb.CallMarket("staticmarket/depth_btc_60.js", nil, nil)
    if rs == nil {
        return nil, nil
    }

    var l int
    ask := make([][]float64, 0, l)
    l = rs.NodeNum("asks")
    for i := 0; i < l; i++ {
        rst := rs.Tree(fmt.Sprintf("asks.%v", i))
        price, _ := rst.Float("0")
        amount, _ := rst.Float("1")
        ask = append(ask, []float64{price, amount})
    }

    bid := make([][]float64, 0, l)
    l = rs.NodeNum("bids")
    for i := 0; i < l; i++ {
        rst := rs.Tree(fmt.Sprintf("bids.%v", i))
        price, _ := rst.Float("0")
        amount, _ := rst.Float("1")
        bid = append(bid, []float64{price, amount})
    }

    return ask, bid
}

func (hb *Huobi) GetBalance() (float64, float64) {
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
    if tree == nil {
        return nil
    }
    if code, has := tree.Int64("code"); has {
        msg, _ := tree.String("message")
        gmvc.Logger.Println(fmt.Sprintf("huobi: %v %s", code, msg))
        return nil
    }
    return tree
}

