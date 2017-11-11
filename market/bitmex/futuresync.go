package bitmex

import (
    "encoding/json"
    "time"
    "strconv"
    "dw/poker/protobuf/exsync"
    "dw/poker/utils"
    "dw/poker/market/common"
)

type FutureSync struct {
    utils.Event
    common.ExCache

    apiKey string
    apiSecret string

    ws *utils.WsClient

    symbol string
}

func NewFutureSync(apiKey, apiSecret, wss, exname string) (*FutureSync, error) {
    var err error
    this := &FutureSync{}
    this.apiKey = apiKey
    this.apiSecret = apiSecret
    this.symbol = "XBTUSD"
    this.Exname = exname

    this.ws = utils.NewWsClient(wss)
    this.ws.AddHandler("Connect", this.connected)
    this.ws.AddHandler("Message", this.newMsg)
    err = this.ws.Start()
    return this, err
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
func (this *FutureSync) connected(args ...interface{}) {
    this.wsauth()
    topics := []interface{}{
        //"chat",        // 聊天室
        //"connected",   // 在线用户/机器人的统计信息
        "instrument:" + this.symbol,  // 产品更新，包括交易量以及报价
        //"insurance",   // 每日保险基金的更新
        //"liquidation", // 强平委托
        //"orderBookL2:XBTUSD", // 完整的 level 2 委托列表
        //"orderBook10:XBTUSD", // 完整的 10 层深度委托列表
        //"publicNotifications", // 通知和告示
        //"quote",       // 报价
        //"quoteBin1m",  // 每分钟报价数据
        //"settlement",  // 结算信息
        "trade:" + this.symbol,       // 实时交易
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

func (this *FutureSync) wsauth() {
    nonce := time.Now().Unix()
    sig := utils.HMAC_SHA256(this.apiSecret, "GET/realtime" + strconv.FormatInt(nonce, 10))
    cmd := wscmd{"authKey", []interface{}{this.apiKey, nonce, sig}}
    this.ws.SendJson(cmd)
}

func (this *FutureSync) newTrade(wsd *wsdata) {
    var resp []tradesResp
    err := json.Unmarshal(wsd.Data, &resp)
    if err != nil {
        utils.WarningLog.Write("bitmex.newTrade %s", err.Error())
    }
    for i := len(resp) - 1; i >= 0; i-- {
        t := resp[i]
        loct := t.Timestamp.Local()
        trade := &exsync.Trade{}
        trade.Id = "bitmex/" + t.TrdMatchID
        trade.CreateTime = &exsync.Timestamp{loct.Unix(), int64(loct.Nanosecond())}
        trade.Price = 1 / t.Price
        trade.Amount = t.Size
        if t.Side == "Sell" {
            trade.TAction = exsync.TradeAction_Sell
        } else {
            trade.TAction = exsync.TradeAction_Buy
        }
        this.NewTrade(trade)
        this.Trigger("NewTrade", trade)
    }
}

func (this *FutureSync) newMsg(args ...interface{}) {
    msg, _ := args[0].([]byte)
    var resp wsresp
    err := json.Unmarshal(msg, &resp)
    if err != nil {
        utils.FatalLog.Write("Exchange.newMsg %s", err.Error())
        return
    }
    if len(resp.Error) > 0 {
        utils.FatalLog.Write("Exchange.newMsg %s", resp.Error)
        return
    }
    if resp.Success {
        utils.DebugLog.Write("Exchange.newMsg %s", resp.Subscribe)
        return
    }

    var wsd wsdata
    err = json.Unmarshal(msg, &wsd)
    if err != nil {
        utils.FatalLog.Write("Exchange.newMsg data %s", err.Error())
        return
    }

    switch wsd.Table {
    case "orderBookL2":
    case "trade":
        this.newTrade(&wsd)

    case "instrument":
        this.instrument(&wsd)

    default:
        utils.WarningLog.Write("topic not handled %s %s", wsd.Table, wsd.Action)
    }
}

type instrumentResp struct {
    MarkPrice float64 `json:"markPrice"`
    IndicativeSettlePrice float64 `json:"indicativeSettlePrice"`
}
func (this *FutureSync) instrument(wsd *wsdata) {
    var ins []instrumentResp
    err := json.Unmarshal(wsd.Data, &ins)
    if err != nil {
        return
    }

    for _, in := range ins {
        if in.MarkPrice > 0 {
            //this.Trigger("IndexUpdate", in.MarkPrice)
        }
        if in.IndicativeSettlePrice > 0 {
            i := 1/in.IndicativeSettlePrice
            this.SetIndex(i)
            this.Trigger("IndexUpdate", i)
        }
    }
}
