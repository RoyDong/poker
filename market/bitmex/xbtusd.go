package bitmex

import (
    "dw/poker/market/utils"
    putils "dw/poker/utils"
    "time"
    "strconv"
    "encoding/json"
    "log"
    "dw/poker/market/context"
)

type Xbtusd struct {
    apiKey string
    apiSecret string
    ws *utils.WsClient


    trades []context.Trade
}

func NewXbtusd(apiKey, apiSecret, wss, host string) (*Xbtusd, error) {
    var err error
    xbt := &Xbtusd{}
    xbt.apiKey = apiKey
    xbt.apiSecret = apiSecret
    xbt.ws = utils.NewWsClient(wss, host, xbt.newMsg, xbt.connected)
    err = xbt.ws.Start()
    return xbt, err
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

func (xbt *Xbtusd) newMsg(msg []byte) {
    var resp wsresp
    err := json.Unmarshal(msg, &resp)
    if err != nil {
        putils.FatalLog.Write("Xbtusd.newMsg %s", err.Error())
        return
    }
    if len(resp.Error) > 0 {
        putils.FatalLog.Write("Xbtusd.newMsg %s", resp.Error)
        return
    }
    if resp.Success {
        putils.DebugLog.Write("Xbtusd.newMsg %s", resp.Subscribe)
        return
    }

    var wsd wsdata
    err = json.Unmarshal(msg, &wsd)
    if err != nil {
        putils.FatalLog.Write("Xbtusd.newMsg data %s", err.Error())
        return
    }
    log.Println(wsd.Table, wsd.Action)

    switch wsd.Table {
    case "orderBookL2":
        xbt.newOrder(&wsd)
    case "trade":
        xbt.newTrade(&wsd)
    default:
        putils.WarningLog.Write("topic not handled %s %s", wsd.Table, wsd.Action)
    }
}

func (xbt *Xbtusd) newTrade(wsd *wsdata) {
}

func (xbt *Xbtusd) newOrder(wsd *wsdata) {

}

func (xbt *Xbtusd) connected() {
    nonce := time.Now().Unix()
    cmd := wscmd{"authKey", []interface{}{xbt.apiKey, nonce, xbt.sign(nonce)}}
    xbt.ws.SendJson(cmd)

    //sub
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
    xbt.ws.SendJson(wscmd{"subscribe", topics})
}

func (xbt *Xbtusd) sign(nonce int64) string {
    return utils.HMAC_SHA256(xbt.apiSecret, "GET/realtime" + strconv.FormatInt(nonce, 10))
}



