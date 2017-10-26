package bitmex

import (
    "time"
    "strconv"
    "encoding/json"
    "log"
    "dw/poker/market/context"
    "dw/poker/market/utils"
    putils "dw/poker/utils"
)

type Exchange struct {
    httpHost string
    apiKey string
    apiSecret string

    ws *utils.WsClient

    symbol string

    trades []context.Trade
}

func NewExchange(httpHost, apiKey, apiSecret, wss, wshost string) (*Exchange, error) {
    var err error
    this := &Exchange{}
    this.httpHost = httpHost
    this.apiKey = apiKey
    this.apiSecret = apiSecret
    this.symbol = "XBTUSD"

    //this.ws = utils.NewWsClient(wss, wshost, this.newMsg, this.connected)
    //err = this.ws.Start()

    return this, err
}

func (this *Exchange) callHttpJson(data interface{}, api string, query, params map[string]interface{}) error {
    host := this.httpHost + api
    resp, err := utils.ReqHttp(host, query, params, this.getAuthHeader(host, query, params))
    if err != nil {
        return err
    }
    return json.Unmarshal(resp, data)
}

func (this *Exchange) getAuthHeader(host string, query, post map[string]interface{}) map[string]interface{} {
    nonce := time.Now().UnixNano() / 1000
    raw := host
    if len(query) > 0 {
        raw = raw + "?" + utils.BuildHttpQuery(query)
    }
    raw = raw + strconv.FormatInt(nonce, 10)
    if len(post) > 0 {
        raw = raw + utils.BuildHttpQuery(post)
    }
    sig := utils.HMAC_SHA256(this.apiSecret, raw)
    return map[string]interface{} {
        "API-NONCE": nonce,
        "API-KEY": this.apiKey,
        "API-SIGNATURE": sig,
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
    Data interface{} `json:"data"`

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

func (this *Exchange) newTrade(wsd *wsdata) {
    log.Println(wsd.Data)
}

func (this *Exchange) newOrder(wsd *wsdata) {

}

func (this *Exchange) connected() {
    nonce := time.Now().Unix()
    sig := utils.HMAC_SHA256(this.apiSecret, "GET/realtime" + strconv.FormatInt(nonce, 10))
    cmd := wscmd{"authKey", []interface{}{this.apiKey, nonce, sig}}
    this.ws.SendJson(cmd)

    topics := []interface{}{
        //"chat",        // 聊天室
        //"connected",   // 在线用户/机器人的统计信息
        //"instrument",  // 产品更新，包括交易量以及报价
        //"insurance",   // 每日保险基金的更新
        //"liquidation", // 强平委托
        "orderBookL2:XBTUSD", // 完整的 level 2 委托列表
        //"orderBook10:XBTUSD", // 完整的 10 层深度委托列表
        //"publicNotifications", // 通知和告示
        //"quote",       // 报价
        //"quoteBin1m",  // 每分钟报价数据
        //"settlement",  // 结算信息
        "trade:XBTUSD",       // 实时交易
        //"tradeBin1m",  // 每分钟交易数据

        //"affiliate",   // 邀请人状态，已邀请用户及分红比率
        "execution",   // 个别成交，可能是多个成交
        "order",       // 你委托的更新
        "margin",      // 你账户的余额和保证金要求的更新
        "position",    // 你仓位的更新
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


/*
func (this *Exchange) CancelOrder(id ...string) error

func (this *Exchange) GetOrder(id string) (context.Order, error)

func (this *Exchange) GetOrders(ids []string) ([]context.Order, error)

func (this *Exchange) GetTicker() (context.Ticker, error)

func (this *Exchange) GetTrades() ([]context.Trade, error)

func (this *Exchange) GetDepth() ([]context.Order, []context.Order, error)

func (this *Exchange) GetIndex() (float64, error)

func (this *Exchange) GetBalance() (context.Balance, error)

func (this *Exchange) GetPosition() (context.Position, context.Position, error)


*/
