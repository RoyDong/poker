package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "io/ioutil"
    "strconv"
    "errors"
    "fmt"
)

type OKCoin struct {
    apiHost   string
    apiKey    string
    apiSecret string
}


func NewOKCoin() *OKCoin {
    conf := gmvc.Store.Tree("config.market.okcoin")
    ok := &OKCoin{}
    ok.apiHost, _ = conf.String("api_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")

    return ok
}


func (ok *OKCoin)Buy(price float64) error {
    p := map[string]interface{}{
        "symbol": "btc_cny",
        "type": "buy_market",
        "price": price,
    }

    rs := ok.Call("trade.do", nil, p)
    if rs == nil {
        return errors.New("okcoin buy error")
    }
    return nil
}


func (ok *OKCoin)Sell(amount float64) error {
    p := map[string]interface{}{
        "symbol": "btc_cny",
        "type": "sell_market",
        "amount": amount,
    }

    rs := ok.Call("trade.do", nil, p)
    if rs == nil {
        return errors.New("okcoin sell error")
    }
    return nil
}


func (ok *OKCoin) LastTicker() *Ticker {
    rs := ok.Call("ticker.do", map[string]interface{}{"symbol": "btc_cny"}, nil)
    if rs == nil {
        return nil
    }

    date, _ := rs.String("date")
    rst     := rs.Tree("ticker")
    high, _ := rst.String("high")
    low,  _ := rst.String("low")
    sell, _ := rst.String("sell")
    buy,  _ := rst.String("buy")
    last, _ := rst.String("last")
    vol,  _ := rst.String("vol")

    t        := &Ticker{}
    t.High, _ = strconv.ParseFloat(high, 10)
    t.Low,  _ = strconv.ParseFloat(low, 10)
    t.Sell, _ = strconv.ParseFloat(sell, 10)
    t.Buy,  _ = strconv.ParseFloat(buy, 10)
    t.Last, _ = strconv.ParseFloat(last, 10)
    t.Vol,  _ = strconv.ParseFloat(vol, 10)
    t.Time, _ = strconv.ParseInt(date, 10, 0)

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
    for i := 0; i < l; i++ {
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

    btc, _ := free.String("btc")
    cny, _ := free.String("cny")

    b, _ := strconv.ParseFloat(btc, 10)
    c, _ := strconv.ParseFloat(cny, 10)

    return b,c
}


func (ok *OKCoin) Call(api string, query, params map[string]interface{}) *gmvc.Tree {
    if params != nil {
        params["api_key"] = ok.apiKey
        params["sign"] = strings.ToUpper(createSignature(params, ok.apiSecret))
    }

    resp, err := CallRest(ok.apiHost + api, query, params)
    if err != nil {
        gmvc.Logger.Println("okcoin: api " + api + "error")
        return nil
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        gmvc.Logger.Println("okcoin: api error")
    }

    tree := gmvc.NewTree()
    err = tree.LoadJson("", body, false)
    if err != nil {
        gmvc.Logger.Println("okcoin: api error not json")
    }

    if _, has := tree.Int("error_code"); has {
        gmvc.Logger.Println("okcoin: api error not json" + string(body))
        return nil
    }

    return tree
}






