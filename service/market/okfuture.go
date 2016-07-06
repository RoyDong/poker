package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "fmt"
    "time"
)

var ExchangeRate float64

type OKFuture struct {

    httpHost   string
    apiKey    string
    apiSecret string

    contractType string
}


func NewOKFuture(contractType string) *OKFuture {
    ok := &OKFuture{}

    conf := gmvc.Store.Tree("config.market.okfuture")
    ok.httpHost, _ = conf.String("http_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")
    ok.contractType = contractType
    ExchangeRate = ok.getExchangeRate()

    return ok
}

func (ok *OKFuture) Buy(price float64) int64 {

    return 0
}

func (ok *OKFuture) Sell(amount float64) int64 {

    return 0
}

func (ok *OKFuture) OrderInfo(id int64) *Order {

    return nil
}

func (ok *OKFuture) syncExchangeRate() {
    for _ = range time.Tick(1 * time.Hour) {
        rate := ok.getExchangeRate()
        if rate > 0 {
            ExchangeRate = rate
        }
    }
}

func (ok *OKFuture) getExchangeRate() float64 {
    rs := ok.Call("exchange_rate.do", nil, nil)
    if rs == nil {
        return 6.5
    }

    rate, _ := rs.Float("rate")
    return rate
}

func (ok *OKFuture) ExchangeRate() float64 {
    return ExchangeRate
}


func (ok *OKFuture) LastTicker() *Ticker {
    q := map[string]interface{}{"symbol": "btc_usd", "contract_type": ok.contractType}
    rs := ok.Call("future_ticker.do", q, nil)
    if rs == nil {
        return nil
    }

    rst := rs.Tree("ticker")
    t := &Ticker{}
    t.High, _ = rst.Float("high")
    t.Low,  _ = rst.Float("low")
    t.Sell, _ = rst.Float("sell")
    t.Buy,  _ = rst.Float("buy")
    t.Last, _ = rst.Float("last")
    t.Vol,  _ = rst.Float("vol")
    t.Time, _ = rs.Int64("date")

    return t
}

func (ok *OKFuture) GetDepth() ([][]float64, [][]float64) {
    query := map[string]interface{}{
        "symbol": "btc_usd",
        "size": 50,
        "merge": 0,
        "contract_type": ok.contractType,
    }

    rs := ok.Call("future_depth.do", query, nil)
    if rs == nil {
        return nil, nil
    }

    var l int
    ask := make([][]float64, 0, l)
    l = rs.NodeNum("asks")
    for i := l - 1; i >= 0; i-- {
        price, _ := rs.Float(fmt.Sprintf("asks.%v.0", i))
        amount, _ := rs.Float(fmt.Sprintf("asks.%v.1", i))
        ask = append(ask, []float64{price, amount})
    }

    bid := make([][]float64, 0, l)
    l = rs.NodeNum("bids")
    for i := 0; i < l; i++ {
        price, _ := rs.Float(fmt.Sprintf("bids.%v.0", i))
        amount, _ := rs.Float(fmt.Sprintf("bids.%v.1", i))
        bid = append(bid, []float64{price, amount})
    }

    return ask, bid
}

func (ok *OKFuture) Index() float64 {
    q := map[string]interface{}{"symbol": "btc_usd"}
    rs := ok.Call("future_index.do", q, nil)
    idx, _ := rs.Float("future_index")
    return idx
}


func (ok *OKFuture) GetBalance() (float64, float64) {
    ok.Call("future_userinfo_4fix.do", nil, map[string]interface{}{})
    return 0, 0
}


func (ok *OKFuture) Call(api string, query, params map[string]interface{}) *gmvc.Tree {
    if params != nil {
        params["api_key"] = ok.apiKey
        params["sign"] = strings.ToUpper(createSignature(params, ok.apiSecret))
    }

    tree := CallRest(ok.httpHost + api, query, params)
    if tree == nil {
        return nil
    }

    if code, has := tree.Int64("error_code"); has {
        gmvc.Logger.Println(fmt.Sprintf("okfuture: %v", code))
        return nil
    }

    return tree
}


