package bitmex

import (
    "time"
    "strconv"
    "encoding/json"
    "log"
    "dw/poker/market/context"
    "dw/poker/market/utils"
    putils "dw/poker/utils"
    "net/url"
    "sync"
)

type Exchange struct {
    httpHost string
    apiKey string
    apiSecret string

    ws *putils.WsClient

    symbol string

    tradesMu sync.RWMutex
    trades []context.Trade
    maxTradesLen int
}

func NewExchange(httpHost, apiKey, apiSecret, wss string) (*Exchange, error) {
    var err error
    this := &Exchange{}
    this.httpHost = httpHost
    this.apiKey = apiKey
    this.apiSecret = apiSecret
    this.symbol = "XBTUSD"
    this.maxTradesLen = 100

    this.trades = make([]context.Trade, 0, this.maxTradesLen)
    this.ws = putils.NewWsClient(wss, this.newMsg, this.connected)
    err = this.ws.Start()

    return this, err
}

func (this *Exchange) callHttpJson(data interface{}, api string, query, params map[string]interface{}, auth bool) error {
    var header map[string]interface{}
    if auth {
        header = this.getAuthHeader(api, query, params)
    }
    resp, err := utils.ReqHttp(this.httpHost + api, query, params, header)
    if err != nil {
        return err
    }
    err = json.Unmarshal(resp, data)
    if err != nil {
        log.Println(string(resp))
    }
    return err
}

func (this *Exchange) getAuthHeader(api string, query, post map[string]interface{}) map[string]interface{} {
    nonce := time.Now().Unix()
    raw := "/api/v1" + api
    if len(query) > 0 {
        raw = raw + "?" + url.QueryEscape(utils.BuildHttpQuery(query))
    }
    if len(post) > 0 {
        raw = "POST" + raw + utils.BuildHttpQuery(post)
    } else {
        raw = "GET" + raw
    }
    sig := utils.HMAC_SHA256(this.apiSecret, raw)
    return map[string]interface{} {
        "api-nonce": nonce,
        "api-key": this.apiKey,
        "api-signature": sig,
    }
}

type wscmd struct {
    Op string `json:"op"`
    Args []interface{} `json:"args"`
}

type wsresp struct {
    Success bool `json:"success"`
    Subscribe string `json:"subscribe"`
    Error string `json:"error"`
}

type wsdata struct {
    Table string `json:"table"`
    Action string `json:"action"`
    Data json.RawMessage `json:"data"`

    Keys []string `json:"keys"`
    ForeignKeys map[string]string `json:"foreignKeys"`

    Types map[string]string `json:"types"`
    Filter map[string]string `json:"filter"`
    Attributes map[string]string `json:"attributes"`
}

func (this *Exchange) newMsg(msg []byte) {
    var resp wsresp
    err := json.Unmarshal(msg, &resp)
    if err != nil {
        putils.FatalLog.Write("Exchange.newMsg %s", err.Error())
        return
    }
    if len(resp.Error) > 0 {
        putils.FatalLog.Write("Exchange.newMsg %s", resp.Error)
        return
    }
    if resp.Success {
        putils.DebugLog.Write("Exchange.newMsg %s", resp.Subscribe)
        return
    }

    var wsd wsdata
    err = json.Unmarshal(msg, &wsd)
    if err != nil {
        putils.FatalLog.Write("Exchange.newMsg data %s", err.Error())
        return
    }

    switch wsd.Table {
    case "orderBookL2":
        this.newOrder(&wsd)
    case "trade":
        this.newTrade(&wsd)
    default:
        putils.WarningLog.Write("topic not handled %s %s", wsd.Table, wsd.Action)
    }
}

type tradesResp struct {
    Timestamp time.Time
    Side string
    Size float64
    Price float64
    TrdMatchID string
}
func (this *Exchange) newTrade(wsd *wsdata) {
    var resp []tradesResp
    err := json.Unmarshal(wsd.Data, &resp)
    if err != nil {
        putils.WarningLog.Write("bitmex.newTrade %s", err.Error())
    }
    trades := make([]context.Trade, 0, len(resp))
    for i := len(resp) - 1; i >= 0; i-- {
        t := resp[i]
        trade := context.Trade{}
        trade.Id = "bitmex/" + t.TrdMatchID
        trade.CreateTime = t.Timestamp.Local()
        trade.Price = 1 / t.Price
        trade.Amount = t.Size
        trade.TAction = context.TradeAction(t.Side)
        trades = append(trades, trade)
    }
    overflow := len(this.trades) + len(trades) - this.maxTradesLen
    if overflow < 0 {
        overflow = 0
    } else if overflow >= this.maxTradesLen {
        overflow = len(this.trades)
    }
    this.tradesMu.Lock()
    this.trades = append(this.trades[overflow:], trades...)
    this.tradesMu.Unlock()
}
func (this *Exchange) GetTrades() ([]context.Trade, error) {
    this.tradesMu.RLock()
    defer this.tradesMu.RUnlock()
    return this.trades, nil
}

func (this *Exchange) newOrder(wsd *wsdata) {

}

func (this *Exchange) wsauth() {
    nonce := time.Now().Unix()
    sig := utils.HMAC_SHA256(this.apiSecret, "GET/realtime" + strconv.FormatInt(nonce, 10))
    cmd := wscmd{"authKey", []interface{}{this.apiKey, nonce, sig}}
    this.ws.SendJson(cmd)
}

func (this *Exchange) connected() {
    this.wsauth()
    topics := []interface{}{
        //"chat",        // 聊天室
        //"connected",   // 在线用户/机器人的统计信息
        //"instrument",  // 产品更新，包括交易量以及报价
        //"insurance",   // 每日保险基金的更新
        //"liquidation", // 强平委托
        //"orderBookL2:XBTUSD", // 完整的 level 2 委托列表
        //"orderBook10:XBTUSD", // 完整的 10 层深度委托列表
        //"publicNotifications", // 通知和告示
        //"quote",       // 报价
        //"quoteBin1m",  // 每分钟报价数据
        //"settlement",  // 结算信息
        "trade:XBTUSD",       // 实时交易
        //"tradeBin1m",  // 每分钟交易数据

        //"affiliate",   // 邀请人状态，已邀请用户及分红比率
        //"execution",   // 个别成交，可能是多个成交
        //"order",       // 你委托的更新
        //"margin",      // 你账户的余额和保证金要求的更新
        //"position",    // 你仓位的更新
        //"privateNotifications", // 个人的通知，现时并未使用
        //"transact",     // 资金提存更新
        //"wallet",       // 比特币余额更新及总提款存款
    }
    this.ws.SendJson(wscmd{"subscribe", topics})
}

func (this *Exchange) Name() string {
    return "bitmex/" + this.symbol
}

func (this *Exchange) GetCurrencyUnit() context.CurrencyUnit {
    return context.BTC
}

type makeOrderReps struct {

}
func (this *Exchange) MakeOrder(ta context.TradeAction, amount, price float64) (context.Order, error) {
    var side, execInst string
    switch ta {
    case context.OpenShort:
        side = "sell"
    case context.OpenLong:
        side = "buy"
    case context.CloseShort:
        side = "sell"
        execInst = "close"
    case context.CloseLong:
        side = "buy"
        execInst = "close"
    default:
        panic("trade action not support")
    }
    params := map[string]interface{} {
        "symbol": this.symbol,
        "side": side,
        "ordType": "limit",
        "orderQty": amount,
        "price": price,
        "execInst": execInst,
    }
    if price <= 0 {
        params["ordType"] = "market"
    }

    var order context.Order
    //this.callHttpJson(d, "/order", nil, params)
    return order, nil
}


func (this *Exchange) CancelOrder(id ...string) error {
    return nil
}

func (this *Exchange) GetOrder(id string) (context.Order, error) {

    return context.Order{}, nil
}

func (this *Exchange) GetOrders(ids []string) ([]context.Order, error) {
    return nil, nil
}

func (this *Exchange) GetTicker() (context.Ticker, error) {
    return context.Ticker{}, nil
}

func (this *Exchange) GetDepth() ([]context.Order, []context.Order, error) {
    return nil,nil,nil
}

func (this *Exchange) GetIndex() (float64, error) {
    return 0, nil
}

func (this *Exchange) GetBalance() (context.Balance, error) {
    var d []byte
    this.callHttpJson(&d, "/user", nil, nil, true)
    return context.Balance{}, nil
}

func (this *Exchange) GetPosition() (context.Position, context.Position, error) {
    return context.Position{}, context.Position{}, nil
}


