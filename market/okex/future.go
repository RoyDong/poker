package okex

import (
    "strings"
    "fmt"
    "time"
    "errors"
    "encoding/json"
    "strconv"
    "dw/poker/market/context"
    "dw/poker/market/utils"
)

type Future struct {
    httpHost   string
    apiKey    string
    apiSecret string

    contractType string
    leverage float64
}

func NewFuture(httpHost, apiKey, apiSecret string) *Future {
    this := &Future{}
    this.httpHost = httpHost
    this.apiKey = apiKey
    this.apiSecret = apiSecret

    this.contractType = "quarter"
    this.leverage = 20
    return this
}

func (this *Future) Name() string {
    return "okex/" + this.contractType
}

type makeOrderResp struct {
    OrderId int64 `json:"order_id"`
    Result bool `json:"result"`
}
func (this *Future) MakeOrder(ta context.TradeAction, amount, price float64) (context.Order, error) {
    price = FutureBTC_USD(price)
    ptype := 0
    switch ta {
    case context.OpenShort:
        ptype = 1
    case context.OpenLong:
        ptype = 2
    case context.CloseShort:
        ptype = 3
    case context.CloseLong:
        ptype = 4
    default:
        panic("trade action not support")
    }
    params := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": this.contractType,
        "type": ptype,
        "amount": fmt.Sprintf("%.0f", amount),
        "price": price,
        "match_price": 0,
        "lever_rate": this.leverage,
    }
    if price <= 0 {
        params["match_price"] = 1
    }
    order := context.Order{}
    mkr := makeOrderResp{}
    err := this.callHttpJson(&mkr, "future_trade.do", nil, params)
    if err != nil {
        return order, err
    }
    if !mkr.Result || mkr.OrderId <= 0 {
        return order, errors.New("make order error")
    }
    order.Id = okidToOrderid(mkr.OrderId)
    return order, nil
}

type getOrderResp struct {
    Orders []struct{
        OrderId int64  `json:"order_id"`
        ContractName string `json:"contract_name"`
        Type int `json:"type"`
        Amount float64 `json:"amount"`
        Price float64 `json:"price"`
        DealAmount float64 `json:"deal_amount"`
        AvgPrice float64 `json:"price_avg"`
        Fee float64 `json:"fee"`
        Status int `json:"status"`
        Leverage int `json:"lever_rate"`
        CreateDate int64 `json:"create_date"`
    } `json:"orders"`
    Result bool `json:"result"`
}

func (this *Future) GetOrder(id string) (context.Order, error) {
    orders, err := this.GetOrders([]string{id})
    if err != nil {
        return context.Order{}, err
    }
    if len(orders) == 0 {
        return context.Order{}, errors.New("no order is found id = " + id)
    }
    return orders[0], err
}

func (this *Future) GetOrders(ids []string) ([]context.Order, error) {
    okids := make([]string, 0, len(ids))
    for _, id := range ids {
        okids = append(okids, fmt.Sprintf("%d", orderidToOkid(id)))
    }
    params := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": this.contractType,
        "order_id": strings.Join(okids, ","),
    }
    resp := getOrderResp{}
    err := this.callHttpJson(&resp, "future_order_info.do", nil, params)
    if err != nil {
        return nil, err
    }
    orders := make([]context.Order, 0, len(ids))
    for _, v := range resp.Orders {
        order := context.Order{}
        order.Id = okidToOrderid(v.OrderId)
        order.Amount = v.Amount
        order.Price = FutureUSD_BTC(v.Price)
        order.DealAmount = v.DealAmount
        order.AvgPrice = FutureUSD_BTC(v.AvgPrice)
        order.Status = v.Status
        order.CreateTime = time.Unix(v.CreateDate, 0)
        orders = append(orders, order)
    }
    return orders, nil
}

type cancelOrderResp struct {
    Result bool `json:"result"`
    OrderId int64 `json:"order_id"`
    Success string `json:"success"`
    Error string `json:"error"`
}
func (this *Future) CancelOrder(ids ...string) error {
    okids := make([]string, 0, len(ids))
    for _, id := range ids {
        okids = append(okids, fmt.Sprintf("%d", orderidToOkid(id)))
    }
    params := map[string]interface{} {
        "symbol": "btc_usd",
        "contract_type": this.contractType,
        "order_id": strings.Join(okids, ","),
    }

    resp := cancelOrderResp{}
    err := this.callHttpJson(&resp, "future_cancel.do", nil, params)
    if err != nil {
        return err
    }
    if !resp.Result || len(resp.Error) > 0 {
        return errors.New("cancel order error")
    }

    return nil
}

type getTradesResp struct {
    Tid int64 `json:"tid"`
    Type string `json:"type"`
    Amount float64 `json:"amount"`
    Price float64 `json:"price"`
    Datems int64 `json:"date_ms"`
}
func (this *Future) GetTrades() ([]context.Trade, error) {
    params := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": this.contractType,
    }

    resp := make([]getTradesResp, 0, 200)
    err := this.callHttpJson(&resp, "future_trades.do", params, nil)
    if err != nil {
        return nil, err
    }

    trades := make([]context.Trade, 0, len(resp))
    for _, t := range resp {
        trade := context.Trade{}
        trade.Id = fmt.Sprintf("okex/%d", t.Tid)
        trade.Amount = t.Amount
        trade.Price = FutureUSD_BTC(t.Price)
        if t.Type == "buy" {
            trade.TAction = context.Buy
        } else {
            trade.TAction = context.Sell
        }
        trade.CreateTime = time.Unix(0, t.Datems * 1e6)
        trades = append(trades, trade)
    }

    return trades, nil
}

type getTickerResp struct {
    Date string `json:"date"`
    Ticker struct {
        Buy float64 `json:"buy"`
        High float64 `json:"high"`
        Last float64 `json:"last"`
        Low float64 `json:"low"`
        Sell float64 `json:"sell"`
        Vol float64 `json:"vol"`
    } `json:"ticker"`
}
func (this *Future) GetTicker() (context.Ticker, error) {
    q := map[string]interface{}{"symbol": "btc_usd", "contract_type": this.contractType}
    resp := getTickerResp{}
    t := context.Ticker{}
    err := this.callHttpJson(&resp, "future_ticker.do", q, nil)
    if err != nil {
        return t, err
    }

    t.High = resp.Ticker.High
    t.Low = resp.Ticker.Low
    t.Ask = resp.Ticker.Sell
    t.Bid = resp.Ticker.Buy
    t.Last = FutureUSD_BTC(resp.Ticker.Last)
    t.Vol = resp.Ticker.Vol
    sec, _ := strconv.ParseInt(resp.Date, 10, 64)
    t.CreateTime = time.Unix(sec, 0)

    return t, nil
}

type getDepthResp struct {
    Asks [][]float64 `json:"asks"`
    Bids [][]float64 `json:"bids"`
}
func (this *Future) GetDepth() ([]context.Order, []context.Order, error) {
    query := map[string]interface{}{
        "symbol": "btc_usd",
        "size": 50,
        "merge": 0,
        "contract_type": this.contractType,
    }

    resp := getDepthResp{}
    err := this.callHttpJson(&resp, "future_depth.do", query, nil)
    if err != nil {
        return nil, nil, err
    }

    /*
    以usd计价转化为以btc计价
    usd ask -> btc bid
    usd bid -> btc ask
     */
    bids := make([]context.Order, 0, len(resp.Asks))
    for _, v := range resp.Asks {
        order := context.Order{}
        order.Amount = v[1]
        order.Price = FutureUSD_BTC(v[0])
        order.AvgPrice = order.Price
        bids = append(bids, order)
    }

    asks := make([]context.Order, 0, len(resp.Bids))
    for _, v := range resp.Bids {
        order := context.Order{}
        order.Amount = v[1]
        order.Price = FutureUSD_BTC(v[0])
        order.AvgPrice = order.Price
        asks = append(asks, order)
    }

    return asks, bids, nil
}

type getIndexResp struct {
    FutureIndex float64 `json:"future_index"`
}
func (this *Future) GetIndex() (float64, error) {
    q := map[string]interface{}{"symbol": "btc_usd"}
    resp := getIndexResp{}
    err := this.callHttpJson(&resp, "future_index.do", q, nil)
    if err != nil {
        return 0, err
    }
    return FutureUSD_BTC(resp.FutureIndex), nil
}


type getBalanceResp struct {
    Info struct {
        Btc struct {
            BalanceRights float64 `json:"account_rights"`
            KeepDeposit float64 `json:"keep_deposit"`
            ProfitReal float64 `json:"profit_real"`
            ProfitUnreal float64 `json:"profit_unreal"`
            RiskRate float64 `json:"risk_rate"`
        } `json:"btc"`

        Ltc struct {
            BalanceRights float64 `json:"account_rights"`
            KeepDeposit float64 `json:"keep_deposit"`
            ProfitReal float64 `json:"profit_real"`
            ProfitUnreal float64 `json:"profit_unreal"`
            RiskRate float64 `json:"risk_rate"`
        } `json:"ltc"`

    } `json:"info"`

    Result bool `json:"result"`
}
func (this *Future) GetBalance() (context.Balance, error) {
    resp := getBalanceResp{}
    err := this.callHttpJson(&resp, "future_userinfo.do", nil, map[string]interface{}{})
    b := context.Balance{}
    if err != nil {
        return b, err
    }
    if !resp.Result {
        return b, errors.New("get account error")
    }

    btc := resp.Info.Btc
    b.Amount = btc.BalanceRights
    b.Deposit = btc.KeepDeposit
    b.RealProfit = btc.ProfitReal
    b.UnrealProfit = btc.ProfitUnreal
    b.RiskRate = btc.RiskRate
    b.Currency = this.GetCurrencyUnit()

    return b, nil
}

type getPositionResp struct {
    ForceLiquPrice string `json:"force_liqu_price"`
    Holding []struct {
        ContractId int64 `json:"contract_id"`
        ContractType string `json:"contract_type"`
        CreateDate int64 `json:"create_date"`

        BuyAmount float64 `json:"buy_amount"`
        BuyAvailable float64 `json:"buy_available"`
        BuyAvgPrice float64 `json:"buy_price_avg"`
        BuyPriceCost float64 `json:"buy_price_cost"`
        BuyRealProfit float64 `json:"buy_profit_real"`

        SellAmount float64 `json:"sell_amount"`
        SellAvailable float64 `json:"sell_available"`
        SellAvgPrice float64 `json:"sell_price_avg"`
        SellPriceCost float64 `json:"sell_price_cost"`
        SellRealProfit float64 `json:"sell_profit_real"`

    } `json:"holding"`
    Result bool `json:"result"`
}
func (this *Future) GetPosition() (context.Position, context.Position, error) {
    p := map[string]interface{}{
        "symbol": "btc_usd",
        "contract_type": this.contractType,
    }
    resp := getPositionResp{}
    err := this.callHttpJson(&resp, "future_position.do", nil, p)
    long := context.Position{PType: context.Long}
    short := context.Position{PType: context.Short}
    if err != nil {
        return long, short, err
    }
    if !resp.Result {
        return long, short, errors.New("get position error")
    }

    for _, holding := range resp.Holding {
        if holding.BuyAmount > 0 {
            short.Id = fmt.Sprintf("okex/%d", holding.ContractId)
            short.Amount = holding.BuyAmount
            short.AvailableAmount = holding.BuyAvailable
            short.AvgPrice = FutureUSD_BTC(holding.BuyAvgPrice)
            short.Deposit = short.Amount * 5 / holding.BuyAvgPrice
            short.Money = short.Deposit
        }
        if holding.SellAmount > 0 {
            long.Id = fmt.Sprintf("okex/%d", holding.ContractId)
            long.Amount = holding.SellAmount
            long.AvailableAmount = holding.SellAvailable
            long.AvgPrice = FutureUSD_BTC(holding.SellAvgPrice)
            long.Deposit = long.Amount * 5 / holding.SellAvgPrice
            long.Money = long.Deposit
        }
    }
    fcp, err := strconv.ParseFloat(strings.Replace(resp.ForceLiquPrice, ",", "", -1), 64)
    fcp = FutureUSD_BTC(fcp)
    long.ForceClosePrice = fcp
    long.Leverage = this.leverage
    short.ForceClosePrice = fcp
    short.Leverage = this.leverage


    return long, short, err
}


func (this *Future) callHttpJson(data interface{}, api string, query, params map[string]interface{}) error {
    if params != nil {
        params["api_key"] = this.apiKey
        params["sign"] = strings.ToUpper(utils.CreateSignature(params, this.apiSecret))
    }
    resp, err := utils.CallRest(this.httpHost + api, query, params)
    if err != nil {
        return err
    }
    return json.Unmarshal(resp, data)
}

func (this *Future) GetCurrencyUnit() context.CurrencyUnit {
    return context.BTC
}

func okidToOrderid(id int64) string {
    return fmt.Sprintf("okex/%d", id)
}

func orderidToOkid(id string) int64 {
    okid, err := strconv.ParseInt(strings.TrimPrefix(id, "okex/"), 10, 64)
    if err != nil {
        panic("not an okex order id")
    }
    return okid
}

/*
得到每一张合约的BTC价格
1 张 = 100USD
 */
func FutureUSD_BTC(usd float64) float64 {
    if usd == 0 {
        return 0
    }
    return 1 / usd * 100
}

func FutureBTC_USD(btc float64) float64 {
    if btc == 0 {
        return 0
    }
    return 1 / btc * 100
}

