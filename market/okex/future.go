package okex

import (
    "strings"
    "fmt"
    "time"
    "errors"
    "encoding/json"
    "strconv"
    "dw/poker/market/common"
    "dw/poker/utils"
    "dw/poker/protobuf/exsync"
    "bytes"
)

type Future struct {
    utils.Event
    common.ExCache
    ws *utils.WsClient

    httpHost   string
    apiKey    string
    apiSecret string
    contractType string
    symbol string
    leverage float64

    tradePipe chan json.RawMessage
    tradeSig chan struct{}
}

func NewFuture(httpHost, wss, apiKey, apiSecret, contractType, exname string) *Future {
    this := &Future{}
    this.httpHost = httpHost
    this.apiKey = apiKey
    this.apiSecret = apiSecret
    this.contractType = contractType
    this.symbol = "btc_usd"
    this.leverage = 20
    this.Exname = exname
    this.TradeLogger = utils.NewLogger("exdata", exname + "-trade", "daily", false)
    this.tradePipe = make(chan json.RawMessage, 10)
    this.tradeSig = make(chan struct{}, 1)

    this.ws = utils.NewWsClient(wss)
    this.ws.AddHandler("Connect", this.connected)
    this.ws.AddHandler("Message", this.newMsg)
    return this
}

func (this *Future) StartSync() error {
    go this.syncTrade()
    go this.syncBalance()
    go this.syncPosition()
    go this.syncPendingOrders()
    return this.ws.Start()
}

func (this *Future) Name() string {
    return "okex/" + this.contractType
}

type makeOrderResp struct {
    OrderId int64 `json:"order_id"`
    Result bool `json:"result"`
}
func (this *Future) MakeOrder(ta exsync.TradeAction, amount, price float64) (*exsync.Order, error) {
    price = FutureBTC_USD(price)
    ptype := 0
    switch ta {
    case exsync.TradeAction_OpenShort:
        ptype = 1
    case exsync.TradeAction_OpenLong:
        ptype = 2
    case exsync.TradeAction_CloseShort:
        ptype = 3
    case exsync.TradeAction_CloseLong:
        ptype = 4
    default:
        panic("trade action not support")
    }
    params := map[string]interface{}{
        "symbol": this.symbol,
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
    order := &exsync.Order{Status:exsync.OrderStatus_Created}
    mkr := makeOrderResp{}
    err := this.callHttpJson(&mkr, "future_trade.do", nil, params)
    if err != nil {
        return order, err
    }
    if !mkr.Result || mkr.OrderId <= 0 {
        return order, errors.New("make order error")
    }
    go this.sendTradeSig()
    order.Id = okidToOrderid(mkr.OrderId)
    this.SetOrder(order)
    this.syncOrder(order)
    return order, nil
}

func (this *Future) sendTradeSig() {
    select {
    case this.tradeSig <- struct{}{}:
    case <-time.After(time.Millisecond):
    }
}

type cancelOrderResp struct {
    Result bool `json:"result"`
    OrderId int64 `json:"order_id"`
    Success string `json:"success"`
    Error string `json:"error"`
}
func (this *Future) CancelOrder(id ...string) error {
    okids := make([]string, 0, len(id))
    for _, id := range id {
        okids = append(okids, fmt.Sprintf("%d", orderidToOkid(id)))
    }
    params := map[string]interface{} {
        "symbol": this.symbol,
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
func (this *Future) GetTicker() (common.Ticker, error) {
    q := map[string]interface{}{"symbol": this.symbol, "contract_type": this.contractType}
    resp := getTickerResp{}
    t := common.Ticker{}
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
func (this *Future) syncBalance() {
    for {
        time.Sleep(10 * time.Second)
        resp := getBalanceResp{}
        err := this.callHttpJson(&resp, "future_userinfo.do", nil, map[string]interface{}{})
        b := &exsync.Balance{}
        if err != nil {
            utils.WarningLog.Write("get balance error %s", err.Error())
            continue
        }
        if !resp.Result {
            utils.WarningLog.Write(this.Exname + " get balance error %v", resp)
            continue
        }

        btc := resp.Info.Btc
        b.Amount = btc.BalanceRights
        b.Deposit = btc.KeepDeposit
        b.RealProfit = btc.ProfitReal
        b.UnrealProfit = btc.ProfitUnreal
        b.RiskRate = btc.RiskRate
        b.Currency = this.GetCurrencyUnit()
        this.SetBalance(b)

        utils.DebugLog.Write(this.Exname + " balance %v", b)
    }
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
func (this *Future) syncPosition() {
    for {
        p := map[string]interface{}{
            "symbol": this.symbol,
            "contract_type": this.contractType,
        }
        resp := getPositionResp{}
        err := this.callHttpJson(&resp, "future_position.do", nil, p)
        long := &exsync.Position{PType: exsync.PositionType_Long}
        short := &exsync.Position{PType: exsync.PositionType_Short}
        if err != nil {
            utils.WarningLog.Write("get position error %s", err.Error())
            time.Sleep(5 * time.Second)
            continue
        }
        if !resp.Result {
            utils.WarningLog.Write(this.Exname + " get position error")
            time.Sleep(5 * time.Second)
            continue
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
        changed := this.SetPosition(long, short)

        utils.DebugLog.Write(this.Exname + " position %v", long)
        utils.DebugLog.Write(this.Exname + " position %v", short)

        if !changed {
            select {
            case <-this.tradeSig:
            case <-time.After(5 * time.Second):
            }
        }
    }
}

func (this *Future) callHttpJson(data interface{}, api string, query, params map[string]interface{}) error {
    if params != nil {
        params["api_key"] = this.apiKey
        params["sign"] = strings.ToUpper(utils.CreateSignature(params, this.apiSecret))
    }
    resp, err := utils.ReqHttp(this.httpHost + api, query, params, nil)
    if err != nil {
        return err
    }
    return json.Unmarshal(resp, data)
}

func (this *Future) GetCurrencyUnit() exsync.CurrencyUnit {
    if this.symbol == "btc_usd" {
        return exsync.CurrencyUnit_BTC
    }
    return exsync.CurrencyUnit_USD
}

func (this *Future) connected(args ...interface{}) {
    channels := []string{
        //最新深度订阅
        fmt.Sprintf("ok_sub_futureusd_btc_depth_%s_%d", this.contractType, 5),
        //最新交易单订阅
        fmt.Sprintf("ok_sub_futureusd_btc_trade_%s", this.contractType),

        "ok_sub_futureusd_btc_index",
    }
    for _, name := range channels {
        this.addChannel(name, nil)
    }
    msg := map[string]interface{} {
        "event": "login",
        "parameters": this.signParams(nil),
    }
    this.ws.SendJson(msg)
}

func (this *Future) addChannel(name string, params map[string]interface{}) {
    msg := map[string]interface{} {
        "event": "addChannel",
        "channel": name,
    }
    if params != nil {
        msg["parameters"] = this.signParams(params)
    }
    this.ws.SendJson(msg)
}

func (this *Future) signParams(params map[string]interface{}) map[string]interface{} {
    if params == nil {
        params = make(map[string]interface{}, 2)
    }
    params["api_key"] = this.apiKey
    params["sign"] = strings.ToUpper(utils.CreateSignature(params, this.apiSecret))
    return params
}

type wsresp struct {
    Channel string  `json:"channel"`
    Data json.RawMessage `json:"data"`
    Result bool `json:"result"`
    ErrorCode int `json:"error_code"`
}
type indexResp struct {
    FutureIndex string `json:"futureIndex"`
    Timestamp string `json:"timestamp"`
}
func (this *Future) newMsg(args ...interface{}) {
    msg, _ := args[0].([]byte)
    var resp []wsresp
    err := json.Unmarshal(msg, &resp)
    if err != nil {
        utils.WarningLog.Write("okex ws sync %s", err.Error())
        return
    }
    if len(resp) == 0 {
        utils.WarningLog.Write("okex ws sync %s", msg)
        return
    }
    for _, r := range resp {
        switch r.Channel {
        case fmt.Sprintf("ok_sub_futureusd_btc_depth_%s_%d", this.contractType, 5):
            this.depthUpdate(r.Data)

        case fmt.Sprintf("ok_sub_futureusd_btc_trade_%s", this.contractType):
            this.tradePipe <-r.Data

        case "ok_sub_futureusd_btc_index":
            var idx indexResp
            json.Unmarshal(r.Data, &idx)
            v, _ := strconv.ParseFloat(idx.FutureIndex, 64)
            i := FutureUSD_BTC(v)
            this.SetIndex(i)
            this.Trigger("IndexUpdate", i)
            utils.DebugLog.Write(this.Exname + " index %f %.8f", v, i)

        case "ok_sub_futureusd_trades":
            this.orderUpdate(r.Data)

        case "ok_futureusd_orderinfo":
            var resp syncOrderResp
            err := json.Unmarshal(r.Data, &resp)
            if err != nil {
                utils.WarningLog.Write(this.Exname + " sync orders err %s", err.Error())
                return
            }
            for _, r := range resp.Orders {
                this.orderUpdate(r)
            }

        case "ok_sub_futureusd_positions":
            //this.positionUpdate(r.Data)

        case "ok_sub_futureusd_userinfo":
            //this.balanceUpdate(r.Data)

        case "btc_forecast_price":
            utils.DebugLog.Write("event %s %s", r.Channel, r.Data)

        case "login", "addChannel":
            utils.DebugLog.Write("event %s", r.Channel)

        default:
            utils.WarningLog.Write("okex channel not handled %s %s", r.Channel, r.Data)
        }
    }
}

func (this *Future) syncTrade() {
    for d := range this.tradePipe {
        var raw [][]string
        err := json.Unmarshal(d, &raw)
        if err != nil {
            utils.WarningLog.Write("ws new trade %s", err.Error())
            continue
        }
        trades := make([]*exsync.Trade, 0, len(raw))
        for _, v := range raw {
            if len(v) == 6 {
                t := &exsync.Trade{}
                t.Id = "okex/" + v[0]
                usd, _ := strconv.ParseFloat(v[1], 64)
                t.Price = FutureUSD_BTC(usd)
                t.Amount, _ = strconv.ParseFloat(v[2], 64)
                if v[4] == "ask" {
                    t.TAction = exsync.TradeAction_Sell
                } else {
                    t.TAction = exsync.TradeAction_Buy
                }
                loct, _ := time.ParseInLocation("15:04:05", v[3], time.Local)
                now := time.Now()
                deltaSec := now.Second() - loct.Second()
                if deltaSec < -30 {
                    deltaSec += 60
                }
                t.CreateTime = &exsync.Timestamp{now.Unix() - int64(deltaSec), 0}
                trades = append(trades, t)
            }
        }
        this.NewTrade(trades)
        this.Trigger("NewTrade", trades)
    }
}


type syncOrderResp struct {
    Result bool `json:"result"`
    Orders []json.RawMessage `json:"orders"`
}
type orderResp struct {
    OrderId int64  `json:"orderid"`
    Order_Id int64  `json:"order_id"`
    ContractName string `json:"contract_name"`
    ContractType string `json:"contract_type"`
    Type int `json:"type"`
    Amount float64 `json:"amount"`
    Price float64 `json:"price"`
    DealAmount float64 `json:"deal_amount"`
    AvgPrice float64 `json:"price_avg"`
    Fee float64 `json:"fee"`
    Status int32 `json:"status"`
    Leverage float64 `json:"lever_rate"`
    CreateDate int64 `json:"create_date"`
}
func (this *Future) orderUpdate(d []byte) {
    var v orderResp
    err := json.Unmarshal(d, &v)
    if err != nil {
        utils.WarningLog.Write("new order error %s", err.Error())
        return
    }
    order := &exsync.Order{}
    if v.OrderId > 0 {
        order.Id = okidToOrderid(v.OrderId)
    } else {
        order.Id = okidToOrderid(v.Order_Id)
    }
    order.Amount = v.Amount
    order.Price = FutureUSD_BTC(v.Price)
    order.DealAmount = v.DealAmount
    order.AvgPrice = FutureUSD_BTC(v.AvgPrice)
    order.Status = exsync.OrderStatus(v.Status)
    switch v.Status {
    case 0:
        order.Status = exsync.OrderStatus_Created
    case 1:
        order.Status = exsync.OrderStatus_Partial
    case 2:
        order.Status = exsync.OrderStatus_Complete
    case -1:
        order.Status = exsync.OrderStatus_Canceled
    case 4:
        order.Status = exsync.OrderStatus_Canceling
    default:
        order.Status = exsync.OrderStatus_StatusUnkown
    }
    order.CreateTime = &exsync.Timestamp{v.CreateDate, 0}
    order.Fee = v.Fee
    switch v.Type {
    case 1:
        order.TAction = exsync.TradeAction_OpenShort
    case 2:
        order.TAction = exsync.TradeAction_OpenLong
    case 3:
        order.TAction = exsync.TradeAction_CloseShort
    case 4:
        order.TAction = exsync.TradeAction_CloseLong
    }
    this.SetOrder(order)
    this.Trigger("OrderUpdate", order)
    utils.DebugLog.Write(this.Exname + " order %v", order.String())
    time.Sleep(time.Millisecond)
    this.syncOrder(order)
}

func (this *Future) syncOrder(order *exsync.Order) {
    if order.Status != exsync.OrderStatus_Complete &&
        order.Status != exsync.OrderStatus_Canceled {

        params := map[string]interface{} {
            "symbol": this.symbol,
            "order_id": orderidToOkid(order.Id),
            "contract_type": this.contractType,
            "current_page": 1,
            "page_length": 50,
        }
        this.addChannel("ok_futureusd_orderinfo", params)
    }
}

func (this *Future) syncPendingOrders() {
    for {
        time.Sleep(5 * time.Second)
        params := map[string]interface{} {
            "symbol": this.symbol,
            "order_id": -1,
            "contract_type": this.contractType,
            "status": 1,
            "current_page": 1,
            "page_length": 50,
        }
        this.addChannel("ok_futureusd_orderinfo", params)
    }
}

type depthResp struct {
    Timestamp int64 `json:"timestamp"`
    Asks json.RawMessage `json:"asks"`
    Bids json.RawMessage `json:"bids"`
}
func (this *Future) depthUpdate(d []byte) {
    var r depthResp
    err := json.Unmarshal(d, &r)
    if err != nil {
        utils.WarningLog.Write("new ws depth error %s", err.Error())
        return
    }
    var asks, bids []*exsync.Trade
    if len(r.Bids) > 2 {
        var rbids [][]float64
        err := json.Unmarshal(bytes.Replace(r.Bids, []byte(`"`), []byte(""), -1), &rbids)
        if err == nil {
            asks = make([]*exsync.Trade, 0, len(rbids))
            for _, v := range rbids {
                row := &exsync.Trade{}
                row.Price = FutureUSD_BTC(v[0])
                row.Amount = v[1]
                row.TAction = exsync.TradeAction_Sell
                row.CreateTime = &exsync.Timestamp{r.Timestamp, 0}
                asks = append(asks, row)
            }
        } else {
            utils.WarningLog.Write("new ws depth error %s", err.Error())
        }
    }
    if len(r.Asks) > 2 {
        var rasks [][]float64
        err := json.Unmarshal(bytes.Replace(r.Asks, []byte(`"`), []byte(""), -1), &rasks)
        if err == nil {
            bids = make([]*exsync.Trade, 0, len(rasks))
            for _, v := range rasks {
                row := &exsync.Trade{}
                row.Price = FutureUSD_BTC(v[0])
                row.Amount = v[1]
                row.TAction = exsync.TradeAction_Buy
                row.CreateTime = &exsync.Timestamp{r.Timestamp, 0}
                bids = append(bids, row)
            }
        } else {
            utils.WarningLog.Write("new ws depth error %s", err.Error())
        }
    }
    this.SetDepth(asks, bids)
    this.Trigger("DepthUpdate", asks, bids)
}

/*
position(string): 仓位 1多仓 2空仓
contract_name(string): 合约名称
costprice(string): 开仓价格
bondfreez(string): 当前合约冻结保证金
avgprice(string): 开仓均价
contract_id(long): 合约id
position_id(long): 仓位id
hold_amount(string): 持仓量
eveningup(string): 可平仓量
margin(double): 固定保证金
realized(double):已实现盈亏
 */
type positionResp struct {
    Symbol string `json:"symbol"`
    Userid int64 `json:"user_id"`
    Positions []struct {
        Id int64 `json:"position_id"`
        Position string `json:"position"`
        ContractName string `json:"contract_name"`
        Margin float64 `json:"margin"`
        Costprice string `json:"costprice"`
        Avgprice string `json:"avgprice"`
        HoldAmount string `json:"hold_amount"`
        Eveningup string `json:"eveningup"`
        Realized float64 `json:"realized"`
    } `json:"positions"`
}
func (this *Future) positionUpdate(d []byte) {
    var r positionResp
    err := json.Unmarshal(d, &r)
    if err != nil {
        utils.WarningLog.Write("okex %s ws position error %s", this.contractType, err.Error())
        return
    }
    if r.Symbol == this.symbol {
        var long, short *exsync.Position
        for _, v := range r.Positions {
            pos := &exsync.Position{}
            pos.Id = fmt.Sprintf("okex/%d", v.Id)
            pos.Amount, _ = strconv.ParseFloat(v.HoldAmount, 64)
            usd, _ := strconv.ParseFloat(v.Avgprice, 64)
            pos.AvgPrice = FutureUSD_BTC(usd)
            pos.Money = pos.Amount * pos.AvgPrice
            pos.AvailableAmount, _ = strconv.ParseFloat(v.Eveningup, 64)
            pos.Deposit = v.Margin
            pos.Leverage = this.leverage
            if v.Position == "2" {
                pos.PType = exsync.PositionType_Long
                long = pos
            } else {
                pos.PType = exsync.PositionType_Short
                short = pos
            }
        }
        this.SetPosition(long, short)
        this.Trigger("PositionUpdate", long, short)
    }
}

/*
{"symbol":"btc_usd","balance":0.15834175,"unit_amount":100.0,"profit_real":-0.00241497,"keep_deposit":0.0028497335}
 */
type balanceResp struct {
    Symbol string `json:"symbol"`
    Balance float64 `json:"balance"`
    UnitAmount float64 `json:"unit_amount"`
    ProfitReal float64 `json:"profit_real"`
    KeepDeposit float64 `json:"keep_deposit"`
}
func (this *Future) balanceUpdate(d []byte) {
    var r balanceResp
    err := json.Unmarshal(d, &r)
    if err != nil {
        utils.WarningLog.Write("okex %s ws sync balance error %s", this.contractType, err.Error())
        return
    }
    if r.Symbol == this.symbol {
        b := &exsync.Balance{}
        b.Amount = r.Balance
        b.Deposit = r.KeepDeposit
        b.RealProfit = r.ProfitReal
        this.SetBalance(b)
        this.Trigger("BalanceUpdate", b)
    }
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

func FutureUSD_LTC(usd float64) float64 {
    if usd == 0 {
        return 0
    }
    return 1 / usd * 10
}

func FutureLTC_USD(ltc float64) float64 {
    if ltc == 0 {
        return 0
    }
    return 1 / ltc * 10
}

