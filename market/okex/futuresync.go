package okex

import (
    "fmt"
    "strings"
    "encoding/json"
    "strconv"
    "dw/poker/protobuf/exsync"
    "time"
    "bytes"
    "log"
    "dw/poker/utils"
    "dw/poker/market/common"
)

type FutureSync struct {
    utils.Event
    common.ExCache

    apiKey string
    apiSecret string
    ws *utils.WsClient

    contractType string
    symbol string
    leverage float64

    tradePipe chan json.RawMessage
}


func NewFutureSync(apiKey, apiSecret, wss, contractType, exname string) (*FutureSync, error) {
    var err error
    this := &FutureSync{}
    this.apiKey = apiKey
    this.apiSecret = apiSecret
    this.contractType = contractType
    this.symbol = "btc_usd"
    this.leverage = 20
    this.Exname = exname
    this.tradePipe = make(chan json.RawMessage, 10)
    go this.syncTrade()

    this.ws = utils.NewWsClient(wss)
    this.ws.AddHandler("Connect", this.connected)
    this.ws.AddHandler("Message", this.newMsg)
    err = this.ws.Start()


    return this, err
}

func (this *FutureSync) connected(args ...interface{}) {
    channels := []string{
        //最新深度订阅
        fmt.Sprintf("ok_sub_futureusd_btc_depth_%s_%d", this.contractType, 5),
        //最新交易单订阅
        fmt.Sprintf("ok_sub_futureusd_btc_trade_%s", this.contractType),

        "ok_sub_futureusd_btc_index",
    }
    for _, name := range channels {
        this.subChannel(name, nil)
    }
    msg := map[string]interface{} {
        "event": "login",
        "parameters": this.signParams(nil),
    }
    this.ws.SendJson(msg)
}

func (this *FutureSync) subChannel(name string, params map[string]interface{}) {
    msg := map[string]interface{} {
        "event": "addChannel",
        "channel": name,
    }
    if params != nil {
        msg["parameters"] = this.signParams(params)
    }
    this.ws.SendJson(msg)
}

func (this *FutureSync) signParams(params map[string]interface{}) map[string]interface{} {
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
    ErrorCode string `json:"error_code"`
}
type indexResp struct {
    FutureIndex string `json:"futureIndex"`
    Timestamp string `json:"timestamp"`
}
func (this *FutureSync) newMsg(args ...interface{}) {
    msg, _ := args[0].([]byte)
    var resp []wsresp
    err := json.Unmarshal(msg, &resp)
    if err != nil {
        utils.WarningLog.Write("okex ws sync %s", err.Error())
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
            this.Trigger("IndexUpdate", FutureUSD_BTC(v))

        case "ok_sub_futureusd_trades":
            this.orderUpdate(r.Data)

        case "ok_sub_futureusd_positions":
            this.positionUpdate(r.Data)

        case "ok_sub_futureusd_userinfo":
            this.balanceUpdate(r.Data)

        case "btc_forecast_price":
            utils.DebugLog.Write("event %s %s", r.Channel, r.Data)

        case "login", "addChannel":
            utils.DebugLog.Write("event %s", r.Channel)

        default:
            utils.WarningLog.Write("okex channel not handled %s %s", r.Channel, r.Data)
        }
    }
}

func (this *FutureSync) syncTrade() {
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


type orderResp struct {
    OrderId int64  `json:"orderid"`
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
func (this *FutureSync) orderUpdate(d []byte) {
    var v orderResp
    err := json.Unmarshal(d, &v)
    if err != nil {
        utils.WarningLog.Write("new order error %s", err.Error())
        return
    }
    if v.ContractType == this.contractType {
        order := &exsync.Order{}
        order.Id = okidToOrderid(v.OrderId)
        order.Amount = v.Amount
        order.Price = FutureUSD_BTC(v.Price)
        order.DealAmount = v.DealAmount
        order.AvgPrice = FutureUSD_BTC(v.AvgPrice)
        order.Status = exsync.OrderStatus(v.Status)
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
    }
}

type depthResp struct {
    Timestamp int64 `json:"timestamp"`
    Asks json.RawMessage `json:"asks"`
    Bids json.RawMessage `json:"bids"`
}
func (this *FutureSync) depthUpdate(d []byte) {
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
func (this *FutureSync) positionUpdate(d []byte) {
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
            pos.Leverge = this.leverage

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
func (this *FutureSync) balanceUpdate(d []byte) {
    log.Println(string(d))
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
        b.RealProfil = r.ProfitReal
        this.SetBalance(b)
        this.Trigger("BalanceUpdate", b)
    }
}


