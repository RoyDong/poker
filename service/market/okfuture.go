package market

import (
    "strings"
    "github.com/roydong/gmvc"
    "io/ioutil"
    "strconv"
    "fmt"
)

type OKFuture struct {

    apiHost   string
    apiKey    string
    apiSecret string

}


func NewOKFuture() *OKFuture {
    ok := &OKFuture{}

    conf := gmvc.Store.Tree("config.market.okfuture")
    ok.apiHost, _ = conf.String("api_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")

    return ok
}


func (ok *OKFuture) Buy(price float64) error {

    return nil
}


func (ok *OKFuture) Sell(amount float64) error {

    return nil
}


func (ok *OKFuture) LastTicker() *Ticker {
    q := map[string]interface{}{"symbol": "btc_usd", "contract_type": "quarter"}
    rs := ok.Call("future_ticker.do", q, nil)
    if rs == nil {
        return nil
    }

    rst := rs["ticker"].(map[string]interface{})
    date, _ := rs["date"].(string)

    t        := &Ticker{}
    t.High, _ = rst["high"].(float64)
    t.Low,  _ = rst["low"].(float64)
    t.Sell, _ = rst["sell"].(float64)
    t.Buy,  _ = rst["buy"].(float64)
    t.Last, _ = rst["last"].(float64)
    t.Vol,  _ = rst["vol"].(float64)
    t.Time, _ = strconv.ParseInt(date, 10, 0)

    return t
}

func (ok *OKFuture) GetDepth() ([][]float64, [][]float64) {
    query := map[string]interface{}{
        "symbol": "btc_usd",
        "size": 50,
        "merge": 0,
        "contract_type": "quarter",
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

func (ok *OKFuture) Index() float64 {
    q := map[string]interface{}{"symbol": "btc_usd"}
    rs := ok.Call("future_index.do", q, nil)
    return rs["future_index"].(float64)
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

    resp, err := CallRest(ok.apiHost + api, query, params)
    if err != nil {
        gmvc.Logger.Println("okfuture: api " + api + "error")
        return nil
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        gmvc.Logger.Println("okfuture: api error")
    }

    tree := gmvc.NewTree()
    err = tree.LoadJson("", body, false)
    if err != nil {
        gmvc.Logger.Println("okfuture: api error not json" + string(body))
        return nil
    }

    if _, has := tree.Int("error_code"); has {
        gmvc.Logger.Println("okfuture: api error")
        return nil
    }

    return tree
}






