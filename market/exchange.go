package market

import (
    "sync"
    "time"
)


type Exchange struct {
    IExchange

    lock sync.RWMutex
    inloop bool
    maxTradeLen int
    trades []Trade
}

func NewExchange(api IExchange) *Exchange {
    ex := &Exchange{
        IExchange: api,
    }

    ex.inloop = true
    ex.maxTradeLen = 1000
    ex.trades = make([]Trade, 0, ex.maxTradeLen)
    go ex.syncTrades()
    return ex
}


func (ex *Exchange) syncTrades() {
    for ex.inloop {
        <- time.After(200 * time.Millisecond)
        trades, err := ex.GetTrades()
        if err != nil {
            continue
        }
        newTrades := make([]Trade, 0, len(trades))
        for _, trade := range trades {
            for i := len(ex.trades); i >= 0; i-- {
                t := ex.trades[i]
                delta := trade.CreateTime.Sub(t.CreateTime)
                if delta > 0 {
                    newTrades = append(newTrades, trade)
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
直接吃掉对手挂单
 */
func (ex *Exchange) TakeDepth(ta TradeAction, amount float64) (Order, error) {
    var price float64
    var err error
    var order Order
    switch ta {
    case OpenLong, CloseShort:
        price, err = ex.GetAskPrice(amount)
    case OpenShort, CloseLong:
        price, err = ex.GetBidPrice(amount)
    }
    if err != nil {
        return order, err
    }
    return ex.MakeOrder(ta, amount, price)
}

