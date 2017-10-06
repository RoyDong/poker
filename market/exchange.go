package market

import (
    "sync"
    "time"
    "dw/poker/utils"
    "dw/poker/market/context"
    "math"
    "errors"
)

type Exchange struct {
    IExchange

    lock sync.RWMutex
    inLoop bool
    maxTradeLen int
    trades []context.Trade
}

func NewExchange(api IExchange) *Exchange {
    ex := &Exchange{
        IExchange: api,
    }

    ex.maxTradeLen = 1000
    ex.inLoop = true
    go ex.syncTrades()
    return ex
}


func (ex *Exchange) syncTrades() {
    if len(ex.trades) <= 0 {
        ex.trades = make([]context.Trade, 1, ex.maxTradeLen)
    }
    for ex.inLoop {
        <- time.After(500 * time.Millisecond)
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
            ex.lock.Lock()
            if overflow := len(ex.trades) + len(newTrades) - ex.maxTradeLen; overflow > 0 {
                ex.trades = ex.trades[overflow:]
            }
            ex.trades = append(ex.trades, newTrades...)
            ex.lock.Unlock()
        }
    }
}

func (ex *Exchange) LastTrade() context.Trade {
    ex.lock.RLock()
    defer ex.lock.RUnlock()
    if l := len(ex.trades); l > 0 {
        return ex.trades[l - 1]
    }
    return context.Trade{}
}

func (ex *Exchange) LastnAvgPrice(n int) float64 {
    ex.lock.RLock()
    defer ex.lock.RUnlock()
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

func (ex *Exchange) OrderCompleteOrPriceChange(order context.Order, spread float64, retry int) (context.Order, bool) {
    for i := 0; i < retry; i++ {
        o, err := ex.GetOrder(order.Id)
        if err != nil {
            utils.WarningLog.Write("sync order error %s", err.Error())
        }
        if o.Status == context.OrderStatusComplete || o.Status == context.OrderStatusCanceled {
            return o, true
        }
        price := ex.LastnAvgPrice(5)
        if math.Abs(price - order.Price) >= spread {
            err = ex.CancelOrder(order.Id)
            if err != nil {
                utils.WarningLog.Write("cancel order error %s", err.Error())
            }
        }
        time.After(100 * time.Millisecond)
    }
    return order, false
}

func (ex *Exchange) Trade(ta context.TradeAction, amount, price float64) (context.Order, error) {
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
    order, ok = ex.OrderCompleteOrPriceChange(order, 10, 10)
    if !ok {
        err = errors.New("exchange trade error Order not complete")
    }
    return order, err
}



