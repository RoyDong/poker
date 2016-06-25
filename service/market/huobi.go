package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "io/ioutil"
    "time"
    "strconv"
    "fmt"
    "errors"
)

type Huobi struct {
    marketHost string
    apiHost    string
    apiKey     string
    apiSecret  string

}


func newHuobi() *Huobi {
    conf := gmvc.Store.Tree("config.market.huobi")
    hb := &Huobi{}
    hb.marketHost, _ = conf.String("market_host")
    hb.apiHost, _ = conf.String("api_host")
    hb.apiKey, _ = conf.String("api_key")
    hb.apiSecret, _ = conf.String("api_secret")

    return hb
}


func (hb *Huobi) Buy(price float64) error {
    q := map[string]interface{}{
        "method": "buy_market",
        "coin_type": 1,
        "amount": fmt.Sprintf("%.2f", price),
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return errors.New("huobi buy error")
    }
    return nil
}


func (hb *Huobi) Sell(amount float64) error {
    q := map[string]interface{}{
        "method": "sell_market",
        "coin_type": 1,
        "amount": amount,
    }

    rs := hb.Call("", nil, q)
    if rs == nil {
        return errors.New("huobi sell error")
    }
    return nil
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
    }

    tree := gmvc.NewTree()
    err = tree.LoadJson("", body, false)
    if err != nil {
        gmvc.Logger.Println("huobi: api error not json" + string(body))
    }

    if code, _ := tree.Float64("code"); code > 0 {
        gmvc.Logger.Println("huobi: api error not json" + string(body))
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
    }

    tree := gmvc.NewTree()
    err = tree.LoadJson("", body, false)
    if err != nil {
        gmvc.Logger.Println("huobi: api error not json" + string(body))
    }

    if code, _ := tree.Float64("code"); code > 0 {
        gmvc.Logger.Println("huobi: api error not json" + string(body))
        return nil
    }

    return tree
}




