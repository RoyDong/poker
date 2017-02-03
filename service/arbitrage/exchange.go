package arbitrage

import (
    "github.com/roydong/gmvc"
    "math"
)


const (
    OpenLongPosition = 1     //多头仓位
    OpenShortPosition = 2    //空头仓位
    CloseLongPosition = 3
    CloseShortPosition = 4
)

type IExchange interface {

    Name() string

    /*
    position 开仓类型 开/平 多/空，对应现货买卖
    amount 数量
    price 价格
    marketPrice 是否以市场价或对手价快速成交
     */
    Trade(position int, amount, price float64, marketPrice bool) Order

    CancelOrder(order Order) bool

    GetTicker() Ticker

    GetTrade() Trade

    GetTrades() []Trade

    GetDepth() ([][]float64, [][]float64)

    GetBalance() (float64, float64)
}

type Ticker struct {
    High float64
    Low  float64
    Ask float64
    Bid  float64
    Last float64
    Vol  float64
    Time int64
}

type Order struct {
    Id int64
    Type int
    Amount float64
    Price float64
    Time int64

    DealAmount float64
    AvgPrice float64
    Status int
    Fee float64
}

type Trade struct {
    Id int64
    Amount float64
    Price float64
    Type string
    Time int64
}

type Exchange struct {
    IExchange

    name string

    amount, money float64

    currency string

    amountChange, moneyChange float64

    fee float64

    lastAsks, lastBids [][]float64

    trades []Trade
    mgm float64
}


func NewExchange(name string) *Exchange {
    e := &Exchange{
        name: name,
        currency: "cny",
    }

    switch e.name {
    case "okcoin":
        e.IExchange = NewOKCoin()
    case "huobi":
        e.IExchange = NewHuobi()
    case "okfuture_quarter":
        e.IExchange = NewOKFutureWS("quarter")
        e.currency = "usd"

    default:
        gmvc.Logger.Fatalln("invalid market " + e.name)
    }

    return e
}

func (e *Exchange) Name() string {
    return e.name
}

/*
获取盘口买单对应深度的价格和数量
 */
func (e *Exchange) GetBidDepth(depth int) (float64, float64) {
    var price, amount float64
    for i, bid := range e.lastBids {
        amount += bid[1]
        if i + 1 >= depth {
            price = bid[0]
            break
        }
    }
    return price, amount
}

/*
获取盘口卖单对应深度的价格和深度
 */
func (e *Exchange) GetAskDepth(depth int) (float64, float64) {
    var price, amount float64
    for i, ask := range e.lastAsks {
        amount += ask[1]
        if i >= depth {
            price = ask[0]
            break
        }
    }
    return price, amount
}

func (e *Exchange) SyncDepth() {
    lastAsks, lastBids := e.GetDepth()
    if len(lastAsks) > 0 && len(lastBids) > 0 {
        e.lastAsks = lastAsks
        e.lastBids = lastBids
    }
}

func (e *Exchange) SyncBalance() {
    e.amount, e.money = e.GetBalance()
}

func (e *Exchange) Balance() (float64, float64) {
    return e.amount, e.money
}

/*
根据最近的交易计算出价格的几何平均数, 以及峰值和峰谷的几何平均数
 */
func (e *Exchange) calcMgm() {
    trades := e.GetTrades()
    if len(trades) < 300 {
        return
    }
    if len(e.trades) > 0 && e.trades[0].Id == trades[0].Id {
        return
    }
    e.trades = trades

    var n, product float64
    n = 0
    product = 1
    for i, trade := range trades {
        if trade.Price > 0 {
            product = product * trade.Price * trade.Amount
            n = n + trade.Amount
        }

        //取最近600次交易
        if i > 600 {
            break
        }
    }
    e.mgm = math.Pow(product, 1 / n)
}

