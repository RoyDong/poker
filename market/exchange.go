package market

import (
    "sync"
    "time"
    "dw/poker/utils"
    "dw/poker/market/context"
    "math"
    "errors"
)

type IExchange interface {
    Name() string

    GetCurrencyUnit() context.CurrencyUnit

    /*
    pos    开仓类型
    amount 数量
    price 价格  price = 0 市价, price = -1 对手价
     */
    MakeOrder(ta context.TradeAction, amount, price float64) (context.Order, error)

    CancelOrder(id ...string) error

    GetOrder(id string) (context.Order, error)

    GetOrders(ids []string) ([]context.Order, error)

    GetTicker() (context.Ticker, error)

    GetTrades() ([]context.Trade, error)

    GetDepth() ([]context.Order, []context.Order, error)

    GetIndex() (float64, error)

    GetBalance() (context.Balance, error)

    GetPosition() (context.Position, context.Position, error)
}

type Exchange struct {
    IExchange

    tradeMu sync.RWMutex
    inLoop bool
    maxTradeLen int
    trades []context.Trade
    kline *context.Kline
}

func NewExchange(api IExchange) *Exchange {
    ex := &Exchange{}
    ex.IExchange = api
    ex.maxTradeLen = 1000
    ex.trades = make([]context.Trade, 1, ex.maxTradeLen)
    ex.inLoop = true
    go ex.syncTrades()
    return ex
}

func (ex *Exchange) syncTrades() {
    for ex.inLoop {
        time.Sleep(200 * time.Millisecond)
        trades, err := ex.GetTrades()
        if err != nil {
            continue
        }
        newTrades := make([]context.Trade, 0, len(trades))
        for _, trade := range trades {
            for i := len(ex.trades) - 1; i >= 0; i-- {
                t := ex.trades[i]
                delta := trade.CreateTime.Sub(t.CreateTime)
                if delta > 0 {
                    newTrades = append(newTrades, trade)
                    utils.DebugLog.Write("new trade: %v", trade)
                    break
                }
                if delta == 0 {
                    if trade.Id == t.Id {
                        break
                    } else {
                        continue
                    }
                }
                if delta < 0 {
                    break
                }
            }
        }
        if len(newTrades) > 0 {
            overflow := len(ex.trades) + len(newTrades) - ex.maxTradeLen
            if overflow < 0 {
                overflow = 0
            }
            ex.tradeMu.Lock()
            ex.trades = append(ex.trades[overflow:], newTrades...)
            ex.tradeMu.Unlock()

            //create kline and save to sql db
            for _, t := range newTrades {
                if ex.kline == nil {
                    ex.kline = context.NewKline(ex.Name(), t, time.Minute)
                } else {
                    rt := ex.kline.AddTrade(t)
                    if rt == 1 {
                        //save
                        utils.Save(ex.kline, "kline", utils.MainDB)
                        ex.kline = context.NewKline(ex.Name(), t, time.Minute)
                    }
                }
            }
        }
    }
}

func (ex *Exchange) LastTrade() context.Trade {
    ex.tradeMu.RLock()
    defer ex.tradeMu.RUnlock()
    if l := len(ex.trades); l > 0 {
        return ex.trades[l - 1]
    }
    return context.Trade{}
}

func (ex *Exchange) LastnAvgPrice(n int) float64 {
    ex.tradeMu.RLock()
    defer ex.tradeMu.RUnlock()
    var sum float64
    if l := len(ex.trades); l > 0 {
        m := l - n
        if m < 0 {
            m = 0
        }
        num := 0
        for i := l - 1; i >= m; i-- {
            sum += ex.trades[i].Price
            num += 1
        }
        return sum / float64(num)
    }
    return 0
}

/*

 */
func (ex *Exchange) GetAskDepth(price float64) (float64, error) {
    asks, _, err := ex.GetDepth()
    if err != nil {
        return 0, err
    }
    var amount float64
    for _, ask := range asks {
        if price >= ask.Price {
            amount += ask.Amount
        } else {
            break
        }
    }
    return amount, nil
}

func (ex *Exchange) GetBidDepth(price float64) (float64, error) {
    _, bids, err := ex.GetDepth()
    if err != nil {
        return 0, err
    }
    var amount float64
    for _, bid := range bids {
        if price <= bid.Price {
            amount += bid.Amount
        } else {
            break
        }
    }
    return amount, nil
}

func (ex *Exchange) GetAskPrice(depth float64) (float64, error) {
    asks, _, err := ex.GetDepth()
    if err != nil {
        return 0, err
    }
    var amount, price float64
    for _, ask := range asks {
        amount += ask.Amount
        price = ask.Price
        if amount >= depth {
            break
        }
    }
    return price, nil
}

func (ex *Exchange) GetBidPrice(depth float64) (float64, error) {
    _, bids, err := ex.GetDepth()
    if err != nil {
        return 0, err
    }
    var amount, price float64
    for _, bid := range bids {
        amount += bid.Amount
        price = bid.Price
        if amount >= depth {
            break
        }
    }
    return price, nil
}

/*
直接吃掉对手挂单指定数量(amount)的深度
快速交易，止损
 */
func (ex *Exchange) TakeDepth(ta context.TradeAction, amount float64) (context.Order, error) {
    var price float64
    var err error
    var order context.Order
    switch ta {
    case context.OpenLong, context.CloseShort, context.Buy:
        price, err = ex.GetAskPrice(amount)
    case context.OpenShort, context.CloseLong, context.Sell:
        price, err = ex.GetBidPrice(amount)
    }
    if err != nil {
        return order, err
    }
    return ex.MakeOrder(ta, amount, price)
}

func (ex *Exchange) OrderCompleteOrPriceChange(order context.Order, spread float64) (context.Order, bool) {
    for i := 0; i < 20; i++ {
        o, err := ex.GetOrder(order.Id)
        if err == nil {
            if o.Status == context.OrderStatusComplete || o.Status == context.OrderStatusCanceled {
                return o, true
            }
            order = o
        }
        price := ex.LastnAvgPrice(5)
        if math.Abs(price - order.Price) >= spread || i >= 15 {
            err = ex.CancelOrder(order.Id)
            if err != nil {
                utils.WarningLog.Write("cancel order error %s", err.Error())
            }
        }
        time.Sleep(100 * time.Millisecond)
    }
    return order, false
}

func (ex *Exchange) Trade(ta context.TradeAction, amount, price, spread float64) (context.Order, error) {
    var order context.Order
    var err error
    if price > 0 {
        order, err = ex.MakeOrder(ta, amount, price)
    } else {
        order, err = ex.TakeDepth(ta, amount)
    }
    if err != nil {
        return order, err
    }
    var ok bool
    order, ok = ex.OrderCompleteOrPriceChange(order, spread)
    if !ok {
        err = errors.New("exchange trade error Order not complete")
    }
    return order, err
}

type Ticker struct {
    Balance context.Balance
    Long context.Position
    Short context.Position
    Price float64
    Index float64
}

func (ex *Exchange) Tick() (Ticker, error) {
    ticker := Ticker{}
    var e1, e2, e3 error
    wg := sync.WaitGroup{}
    wg.Add(3)
    go func() {
        ticker.Balance, e1 = ex.GetBalance()
        wg.Done()
    }()
    go func() {
        ticker.Long, ticker.Short, e2 = ex.GetPosition()
        wg.Done()
    }()
    go func() {
        ticker.Index, e3 = ex.GetIndex()
        wg.Done()
    }()
    wg.Wait()
    if e1 != nil {
        return ticker, e1
    }
    if e2 != nil {
        return ticker, e2
    }
    if e3 != nil {
        return ticker, e3
    }
    ticker.Price = ex.LastnAvgPrice(5)
    return ticker, nil
}

