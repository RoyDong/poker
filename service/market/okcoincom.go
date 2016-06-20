package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "encoding/json"
    "io/ioutil"
    "strconv"
)

type OKCoinCom struct {

    apiHost   string
    apiKey    string
    apiSecret string

}


func NewOKCoinCom() *OKCoinCom {
    ok := &OKCoinCom{}

    conf := gmvc.Store.Tree("config.market.okcoincom")
    ok.apiHost, _ = conf.String("api_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")

    return ok
}


func (ok *OKCoinCom)Buy() {

}


func (ok *OKCoinCom)Sell() {

}


func (ok *OKCoinCom)Ticker() Ticker {
    q := map[string]interface{}{"symbol": "btc_usd", "contract_type": "quarter"}
    rs := ok.Call("future_ticker.do", q, nil)
    t := Ticker{}

    date, _ := rs["date"].(string)

    rs = rs["ticker"].(map[string]interface{})
    t.High, _ = rs["high"].(float64)
    t.Low,  _ = rs["low"].(float64)
    t.Sell, _ = rs["sell"].(float64)
    t.Buy,  _ = rs["buy"].(float64)
    t.Last, _ = rs["last"].(float64)
    t.Vol,  _ = rs["vol"].(float64)

    t.Time, _ = strconv.ParseInt(date, 10, 0)

    return t
}

func (ok *OKCoinCom)Index() float64 {
    q := map[string]interface{}{"symbol": "btc_usd"}
    rs := ok.Call("future_index.do", q, nil)
    return rs["future_index"].(float64)
}


func (ok *OKCoinCom)UserInfo() interface{} {
    q := map[string]interface{}{}
    return ok.Call("userinfo.do", nil, q)
}


func (ok *OKCoinCom)Call(api string, query, params map[string]interface{}) map[string]interface{} {
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
        gmvc.Logger.Println("okcoin: api error not json")
    }

    return rs
}






