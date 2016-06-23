package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "encoding/json"
    "io/ioutil"
    "strconv"
)

type OKCoin struct {

    apiHost   string
    apiKey    string
    apiSecret string

}


func NewOKCoin() *OKCoin {
    ok := &OKCoin{}

    conf := gmvc.Store.Tree("config.market.okcoin")
    ok.apiHost, _ = conf.String("api_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")

    return ok
}


func (ok *OKCoin)Buy() {

}


func (ok *OKCoin)Sell() {

}


func (ok *OKCoin)Ticker() Ticker {
    q := map[string]interface{}{"symbol": "btc_cny"}
    rs := ok.Call("ticker.do", q, nil)
    t := Ticker{}

    date, _ := rs["date"].(string)

    rs = rs["ticker"].(map[string]interface{})
    high, _ := rs["high"].(string)
    low,  _ := rs["low"].(string)
    sell, _ := rs["sell"].(string)
    buy,  _ := rs["buy"].(string)
    last, _ := rs["last"].(string)
    vol,  _ := rs["vol"].(string)

    t.High, _ = strconv.ParseFloat(high, 10)
    t.Low,  _ = strconv.ParseFloat(low, 10)
    t.Sell, _ = strconv.ParseFloat(sell, 10)
    t.Buy,  _ = strconv.ParseFloat(buy, 10)
    t.Last, _ = strconv.ParseFloat(last, 10)
    t.Vol,  _ = strconv.ParseFloat(vol, 10)
    t.Time, _ = strconv.ParseInt(date, 10, 0)

    return t
}

/*
func (ok *OKCoin)SyncTicker(interval time.Millisecond) {
    for t := time.Tick(interval) {
        ticker := ok.Ticker()

    }
}
*/

func (ok *OKCoin)UserInfo() interface{} {
    q := map[string]interface{}{}
    return ok.Call("userinfo.do", nil, q)
}


func (ok *OKCoin)Call(api string, query, params map[string]interface{}) map[string]interface{} {
    if params != nil {
        params["api_key"] = ok.apiKey
        params["secret_key"] = ok.apiSecret
        params["sign"] = strings.ToUpper(createSignature(params))
        delete(params, "secret_key")
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

    var rs map[string]interface{}
    err = json.Unmarshal(body, &rs)
    if err != nil {
        gmvc.Logger.Println("okcoin: api error not json" + string(body))
    }

    return rs
}






