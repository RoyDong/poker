package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "strconv"
    "fmt"
    "time"
)

var ExchangeRate float64

type OKFuture struct {

    apiHost   string
    apiKey    string
    apiSecret string

    contractType string
}


func NewOKFuture(contractType string) *OKFuture {
    ok := &OKFuture{}

    conf := gmvc.Store.Tree("config.market.okfuture")
    ok.apiHost, _ = conf.String("api_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")
    ok.contractType = contractType
    ExchangeRate = ok.getExchangeRate()

    //go ok.syncExchangeRate()

    return ok
}

func (ok *OKFuture) Buy(price float64) error {

    return nil
}

func (ok *OKFuture) Sell(amount float64) error {

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

    rate, _ := rs.Float64("rate")
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
    t.High, _ = rst.Float64("high")
    t.Low,  _ = rst.Float64("low")
    t.Sell, _ = rst.Float64("sell")
    t.Buy,  _ = rst.Float64("buy")
    t.Last, _ = rst.Float64("last")
    t.Vol,  _ = rst.Float64("vol")

    t.Last = t.Last *ExchangeRate

    date, _  := rs.String("date")
    t.Time, _ = strconv.ParseInt(date, 10, 0)

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
        price, _ := rs.Float64(fmt.Sprintf("asks.%v.%v", i, 0))
        amount, _ := rs.Float64(fmt.Sprintf("asks.%v.%v", i, 1))
        ask = append(ask, []float64{price * ExchangeRate, amount})
    }

    bid := make([][]float64, 0, l)
    l = rs.NodeNum("bids")
    for i := 0; i < l; i++ {
        price, _ := rs.Float64(fmt.Sprintf("bids.%v.%v", i, 0))
        amount, _ := rs.Float64(fmt.Sprintf("bids.%v.%v", i, 1))
        bid = append(bid, []float64{price *ExchangeRate, amount})
    }

    return ask, bid
}

func (ok *OKFuture) Index() float64 {
    q := map[string]interface{}{"symbol": "btc_usd"}
    rs := ok.Call("future_index.do", q, nil)
    idx, _ := rs.Float64("future_index")
    return idx
}


func (ok *OKFuture) GetBalance() (float64, float64) {
    rs := ok.Call("future_userinfo_4fix.do", nil, map[string]interface{}{})
    return 0, 0
    free := rs.Tree("info.funds.free")
    if free == nil {
        return 0, 0
    }

    btc, _ := free.String("btc")
    cny, _ := free.String("cny")

    b, _ := strconv.ParseFloat(btc, 10)
    c, _ := strconv.ParseFloat(cny, 10)

    return b,c
}


func (ok *OKFuture)Call(api string, query, params map[string]interface{}) *gmvc.Tree {
    if params != nil {
        params["api_key"] = ok.apiKey
        params["sign"] = strings.ToUpper(createSignature(params, ok.apiSecret))
    }

    tree := CallRest(ok.apiHost + api, query, params)
    if code, has := tree.Int64("error_code"); has {
        gmvc.Logger.Println(fmt.Sprintf("okfuture: %v", code))
        return nil
    }

    return tree
}






