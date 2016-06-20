package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "encoding/json"
    "io/ioutil"
    "time"
    "strconv"
)

type Huobi struct {

    marketHost string
    apiHost    string
    apiKey     string
    apiSecret  string

}


func NewHuobi() *Huobi {
    hb := &Huobi{}

    conf := gmvc.Store.Tree("config.market.huobi")
    hb.marketHost, _ = conf.String("market_host")
    hb.apiHost, _ = conf.String("api_host")
    hb.apiKey, _ = conf.String("api_key")
    hb.apiSecret, _ = conf.String("api_secret")

    return hb
}


func (hb *Huobi)Buy() {

}


func (hb *Huobi)Sell() {

}


func (hb *Huobi)Ticker() Ticker {
    rs := hb.CallMarket("staticmarket/ticker_btc_json.js", nil, nil)

    t := Ticker{}

    time, _ := rs["time"].(string)

    rs = rs["ticker"].(map[string]interface{})
    t.High, _ = rs["high"].(float64)
    t.Low,  _ = rs["low"].(float64)
    t.Sell, _ = rs["sell"].(float64)
    t.Buy,  _ = rs["buy"].(float64)
    t.Last, _ = rs["last"].(float64)
    t.Vol,  _ = rs["vol"].(float64)
    t.Time, _ = strconv.ParseInt(time, 10, 0)

    return t
}


func (hb *Huobi)UserInfo() interface{} {
    q := map[string]interface{}{}
    return hb.Call("userinfo.do", nil, q)
}


func (hb *Huobi)Call(api string, query, params map[string]interface{}) map[string]interface{} {
    if params != nil {
        params["access_key"] = hb.apiKey
        params["secret_key"] = hb.apiSecret
        params["created"] = time.Now().Unix()
        params["sign"] = strings.ToLower(createSignature(params))
        delete(params, "secret_key")
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

    var rs map[string]interface{}
    err = json.Unmarshal(body, &rs)
    if err != nil {
        gmvc.Logger.Println("huobi: api error not json" + string(body))
    }

    gmvc.Logger.Println(string(body))

    return rs
}


func (hb *Huobi)CallMarket(api string, query, params map[string]interface{}) map[string]interface{} {
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

    var rs map[string]interface{}
    err = json.Unmarshal(body, &rs)
    if err != nil {
        gmvc.Logger.Println("huobi: api error not json" + string(body))
    }

    return rs
}




