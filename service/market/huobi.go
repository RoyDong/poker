package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "io/ioutil"
    "time"
    "strconv"
    "fmt"
)

type Huobi struct {
    marketHost string
    apiHost    string
    apiKey     string
    apiSecret  string

}


func NewHuobi() *Huobi {
    conf := gmvc.Store.Tree("config.market.huobi")
    hb := &Huobi{}
    hb.marketHost, _ = conf.String("market_host")
    hb.apiHost, _ = conf.String("api_host")
    hb.apiKey, _ = conf.String("api_key")
    hb.apiSecret, _ = conf.String("api_secret")

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

    amount,     _ := rs.String("order_amount")
    price,      _ := rs.String("order_price")
    dealAmount, _ := rs.String("processed_amount")
    avgPrice,   _ := rs.String("processed_price")

    order.Amount,     _ = strconv.ParseFloat(amount, 10)
    order.Price,      _ = strconv.ParseFloat(price, 10)
    order.DealAmount, _ = strconv.ParseFloat(dealAmount, 10)
    order.AvgPrice,   _ = strconv.ParseFloat(avgPrice, 10)

    typ, _ := rs.Float64("type")
    if int64(typ) == 3 {
        order.Price = order.Amount
        order.Amount = 0
    }

    order.Created = time.Now().Unix()

    return order
}


func (hb *Huobi) LastTicker() *Ticker {
    rs := hb.CallMarket("staticmarket/ticker_btc_json.js", nil, nil)
    rst := rs.Tree("ticker")
    t := &Ticker{}
    t.High, _ = rst.Float64("high")
    t.Low,  _ = rst.Float64("low")
    t.Sell, _ = rst.Float64("sell")
    t.Buy,  _ = rst.Float64("buy")
    t.Last, _ = rst.Float64("last")
    t.Vol,  _ = rst.Float64("vol")

    time, _ := rs.String("time")
    t.Time, _ = strconv.ParseInt(time, 10, 0)

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
    btc, _ := rs.String("available_btc_display")
    cny, _ := rs.String("available_cny_display")

    b, _ := strconv.ParseFloat(btc, 10)
    c, _ := strconv.ParseFloat(cny, 10)

    return b,c
}


func (hb *Huobi) Call(api string, query, params map[string]interface{}) *gmvc.Tree {
    if params != nil {
        params["access_key"] = hb.apiKey
        params["created"] = time.Now().Unix()
        params["sign"] = strings.ToLower(createSignature(params, hb.apiSecret))
    }

    resp, err := CallRest(hb.apiHost + api, query, params)
    if err != nil {
        gmvc.Logger.Println("huobi: api " + api + "error")
        return nil
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        gmvc.Logger.Println("huobi: api error")
        return nil
    }

    tree := gmvc.NewTree()
    err = tree.LoadJson("", body, false)
    if err != nil {
        gmvc.Logger.Println("huobi: api error not json")
        return nil
    }

    if tree.Get("code") != nil {
        gmvc.Logger.Println("huobi: api error" + string(body))
        return nil
    }

    return tree
}


func (hb *Huobi) CallMarket(api string, query, params map[string]interface{}) *gmvc.Tree {
    resp, err := CallRest(hb.marketHost + api, query, params)
    if err != nil {
        gmvc.Logger.Println("huobi: api " + api + "error")
        return nil
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        gmvc.Logger.Println("huobi: api error")
        return nil
    }

    tree := gmvc.NewTree()
    err = tree.LoadJson("", body, false)
    if err != nil {
        gmvc.Logger.Println("huobi: api error not json")
        return nil
    }

    if tree.Get("code") != nil {
        gmvc.Logger.Println("huobi: api error" + string(body))
        return nil
    }

    return tree
}




