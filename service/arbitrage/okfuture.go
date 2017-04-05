package arbitrage

import (
    "strings"
    "github.com/roydong/gmvc"
    "fmt"
)

type OKFuture struct {

    httpHost   string
    apiKey    string
    apiSecret string

    contractType string
    leverRate int
}


func NewOKFuture(contractType string, leverRate int) *OKFuture {
    ok := &OKFuture{}

    conf := gmvc.Store.Tree("config.market.okfuture")
    ok.httpHost, _ = conf.String("http_host")
    ok.apiKey, _ = conf.String("api_key")
    ok.apiSecret, _ = conf.String("api_secret")
    ok.contractType = contractType
    ok.leverRate = leverRate

    return ok
}

func (ok *OKFuture) Trade(position int, amount, price float64) int64 {
    params := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": ok.contractType,
        "type": position,
        "amount": fmt.Sprintf("%.0f", amount),
        "price": price,
        "match_price": 0,
        "lever_rate": ok.leverRate,
    }
    if price == 0 {
        params["match_price"] = 1
    }
    rs := ok.Call("future_trade.do", nil, params)
    if rs == nil {
        return 0
    }
    id, _ := rs.Int64("order_id")
    return id
}

func (ok *OKFuture) Order(id int64) Order {
    params := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": ok.contractType,
        "order_id": id,
    }

    order := Order{}
    rs := ok.Call("future_order_info.do", nil, params)
    if rs == nil {
        return order
    }

    rst := rs.Tree("orders.0")
    if rst == nil {
        return order
    }

    order.Id = id
    order.Amount, _ = rst.Float("amount")
    order.Price, _ = rst.Float("price")
    order.DealAmount, _ = rst.Float("deal_amount")
    order.AvgPrice, _ = rst.Float("price_avg")
    order.Status, _ = rst.Int("status")
    t, _ := rst.Float("create_date")
    order.Time = int64(t)

    return order
}

func (ok *OKFuture) CancelOrder(id int64) bool {
    params := map[string]interface{} {
        "symbol": "btc_usd",
        "contract_type": ok.contractType,
        "order_id": id,
    }

    rs := ok.Call("future_cancel.do", nil, params)
    if rs == nil {
        return false
    }

    if result, _ := rs.String("result"); result == "true" {
        return true
    }

    return false
}

func (ok *OKFuture) GetExchangeRate() float64 {
    rs := ok.Call("exchange_rate.do", nil, nil)
    if rs == nil {
        return 6.8
    }

    rate, _ := rs.Float("rate")
    return rate
}

func (ok *OKFuture) GetTrades() []Trade {
    params := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": ok.contractType,
    }

    rs := ok.Call("future_trades.do", params, nil)
    if rs == nil {
        return nil
    }

    n := rs.NodeNum("")
    trades := make([]Trade, 0, n)
    for i := 0; i < n; i++ {
        trade := Trade{}
        trade.Id, _ = rs.Int64(fmt.Sprintf("%d.tid", i))
        trade.Price, _ = rs.Float(fmt.Sprintf("%d.price", i))
        trade.Amount, _ = rs.Float(fmt.Sprintf("%d.amount", i))
        trade.Time, _ = rs.Int64(fmt.Sprintf("%d.date_ms", i))
        trade.Type, _ = rs.String(fmt.Sprintf("%d.type", i))
        trades = append(trades, trade)
    }
    return trades
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
    t.Ask, _ = rst.Float("sell")
    t.Bid,  _ = rst.Float("buy")
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


func (ok *OKFuture) GetBalance() Balance {
    var balance Balance
    rs := ok.Call("future_userinfo.do", nil, map[string]interface{}{})
    if rs == nil {
        return balance
    }

    btcInfo := rs.Tree("info.btc")
    if btcInfo == nil {
        return balance
    }

    balance.AccountRights, _ = btcInfo.Float("account_rights")
    balance.Deposit, _ = btcInfo.Float("keep_deposit")

    params := map[string]interface{} {
        "symbol": "btc_usd",
        "contract_type": ok.contractType,
    }
    rs = ok.Call("future_position.do", nil, params)
    if rs == nil {
        return balance
    }

    holding := rs.Tree("holding.0")
    if holding == nil {
        return balance
    }

    balance.ContractId , _ = holding.Int64("contract_id")
    balance.LongAmount, _ = holding.Float("buy_amount")
    balance.LongPrice, _ = holding.Float("buy_price_cost")
    balance.LongProfit, _ = holding.Float("buy_profit_real")

    balance.ShortAmount, _ = holding.Float("sell_amount")
    balance.ShortPrice, _ = holding.Float("sell_price_avg")
    balance.ShortProfit, _ = holding.Float("sell_profit_real")

    return balance
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


