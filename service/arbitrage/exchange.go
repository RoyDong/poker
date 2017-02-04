package arbitrage

import (
    "github.com/roydong/gmvc"
    //"math"
    "time"
    "fmt"
    "log"
    "container/list"
)


const (
    OpenLongPosition = 1     //多头仓位
    OpenShortPosition = 2    //空头仓位
    CloseLongPosition = 3
    CloseShortPosition = 4


    OrderStatusCreated   = 0
    OrderStatusPartial   = 1
    OrderStatusComplete  = 2
    OrderStatusCancel    = -1
    OrderStatusCanceling = 4
)

type IExchange interface {

    /*
    position 开仓类型 开/平 多/空，对应现货买卖
    amount 数量
    price 价格  如果price = 0 则为对手价
     */
    Trade(position int, amount, price float64) int64

    Order(id int64) Order

    CancelOrder(id int64) bool

    //GetTicker() Ticker

    //GetTrade() Trade

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

    trades *list.List
    ma float64
}


func NewExchange(name string) *Exchange {
    e := &Exchange{
        name: name,
        currency: "cny",
        trades: list.New(),
    }

    switch e.name {
    case "okcoin":
        //e.IExchange = NewOKCoin()
    case "huobi":
        //e.IExchange = NewHuobi()
    case "okfuture_quarter":
        e.IExchange = NewOKFuture("quarter", 20)
        e.currency = "usd"
    case "okfuture_thisweek":
        e.IExchange = NewOKFuture("this_week", 20)
        e.currency = "usd"

    default:
        gmvc.Logger.Fatalln("invalid exchange " + e.name)
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
根据最近的交易计算出价格的几何平均数
 */
func (e *Exchange) calcMa() {
    trades := e.IExchange.GetTrades()
    var trade Trade
    if e.trades.Len() > 1 {
        trade, _ = e.trades.Front().Value.(Trade)
    }

    for _, t := range trades {
        if t.Id > trade.Id {
            e.trades.PushFront(t)
            if e.trades.Len() > 600 {
                e.trades.Remove(e.trades.Back())
            }
        }
    }

    var n, sum float64
    for el := e.trades.Back(); el != nil; el = el.Prev() {
        trade, _ := el.Value.(Trade)
        if trade.Price > 0 {
            sum = sum + trade.Price * trade.Amount
            n = n + trade.Amount
        }
    }
    e.ma = sum / n
    log.Println(e.Name(), e.ma)
}

/*
price = 0 对手价
 */
func (e *Exchange) Trade(position int, amount, price float64) Order {
    var order Order
    var id int64
    for i := 0; i < 3; i++ {
        id = e.IExchange.Trade(position, amount, price)
        if id > 0 {
            break
        }
    }
    if id > 0 {
        //每隔0.5s读取一次，最多等待5s
        for i := 0; i < 10; i++ {
            time.Sleep(500 * time.Millisecond)
            order = e.IExchange.Order(id)
            if order.Status == 2 {
                break
            }
        }

        //如果订单没有完全成交
        if order.Status != 2 {
            canceled := false

            //重试两次，如果都失败中断程序
            for i := 0; i < 2; i++ {
                canceled = e.IExchange.CancelOrder(id)
            }

            if !canceled {
                gmvc.Logger.Println(fmt.Sprintf("cancel order failed %v order id = %v", e.Name(), id))
            }

            //更新order info
            for i := 0; i < 3; i++ {
                order = e.IExchange.Order(id)
                if order.Id > 0 {
                    break
                }
            }

            if order.Id == 0 {
                gmvc.Logger.Println(fmt.Sprintf("update order info failed %v order id = %v", e.Name(), id))
            }
        }
    } else {
        gmvc.Logger.Println(fmt.Sprintf("make order failed %v", e.Name()))
    }
    return order
}

/*
不断以对手价下单直到交易完amount数量
 */
func (e *Exchange) TradeAll(position int, amount float64) Order {
    var order Order
    var money float64
    for order.DealAmount < amount {
        o := e.Trade(position, amount - order.DealAmount, 0)
        if order.Id <= 0 {
            order.Id = o.Id
        }
        money += o.DealAmount * o.AvgPrice
        order.DealAmount += o.DealAmount
    }
    order.AvgPrice = money / order.DealAmount
    return order
}

