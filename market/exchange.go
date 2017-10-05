package market

import (
    "sync"
    "time"
    "github.com/roydong/poker/utils"
    "github.com/roydong/poker/market/base"
)


type Exchange struct {
    IExchange

    lock sync.RWMutex
    inLoop bool
    maxTradeLen int
    trades []base.Trade
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
        ex.trades = make([]base.Trade, 1, ex.maxTradeLen)
    }
    for ex.inLoop {
        <- time.After(500 * time.Millisecond)
        trades, err := ex.GetTrades()
        if err != nil {
            continue
        }
        newTrades := make([]base.Trade, 0, len(trades))
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

func (ex *Exchange) LastTrade() base.Trade {
    ex.lock.RLock()
    defer ex.lock.RUnlock()
    return ex.trades[len(ex.trades) - 1]
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
func (ex *Exchange) TakeDepth(ta base.TradeAction, amount float64) (base.Order, error) {
    var price float64
    var err error
    var order base.Order
    switch ta {
    case base.OpenLong, base.CloseShort, base.Buy:
        price, err = ex.GetAskPrice(amount)
    case base.OpenShort, base.CloseLong, base.Sell:
        price, err = ex.GetBidPrice(amount)
    }
    if err != nil {
        return order, err
    }
    return ex.MakeOrder(ta, amount, price)
}

func (ex *Exchange) SyncOrder(order base.Order, retry int) (base.Order, bool) {
    for i := 0; i < retry; i++ {
        time.After(200 * time.Millisecond)
        o, err := ex.GetOrder(order.Id)
        if err == nil {
            if o.DealAmount > order.DealAmount {
                return o, true
            }
            if o.Status != order.Status {
                return o, true
            }
        } else {
            utils.WarningLog.Write("sync order error %s", err.Error())
        }
    }
    return order, false
}


