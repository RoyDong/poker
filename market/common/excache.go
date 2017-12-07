package common

import (
    "sync"
    "dw/poker/protobuf/exsync"
    "time"
    "dw/poker/utils"
)

type ExCache struct {
    Exname string
    mu sync.RWMutex
    trades []*exsync.Trade

    index float64

    asks []*exsync.Trade
    bids []*exsync.Trade

    orders map[string]*exsync.Order

    long *exsync.Position
    short *exsync.Position

    balance *exsync.Balance

    kline *Kline
    klines []*Kline

    tradeLogger *utils.Logger
}

func NewExCache(exname string) *ExCache {
    c := &ExCache{}
    c.Exname = exname
    c.balance = &exsync.Balance{}
    c.long = &exsync.Position{}
    c.short = &exsync.Position{}
    c.asks = make([]*exsync.Trade, 0)
    c.bids = make([]*exsync.Trade, 0)
    c.orders = make(map[string]*exsync.Order, 10)
    c.tradeLogger = utils.NewLogger("exdata", exname + "-trade", "daily", false)
    return c
}

func (c *ExCache) SetBalance(b *exsync.Balance) {
    c.mu.Lock()
    c.balance = b
    c.mu.Unlock()
}

func (c *ExCache) GetBalance() *exsync.Balance {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.balance
}

func (c *ExCache) SetPosition(long, short *exsync.Position) bool {
    changed := false
    c.mu.Lock()
    if c.long.Amount != long.Amount {
        c.long = long
        changed = true
    }
    if c.short.Amount != short.Amount {
        c.short = short
        changed = true
    }
    c.mu.Unlock()
    return changed
}

func (c *ExCache) GetLong() *exsync.Position {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.long
}

func (c *ExCache) GetShort() *exsync.Position {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.short
}

func (c *ExCache) SetOrder(order *exsync.Order) {
    c.mu.Lock()
    c.orders[order.GetId()] = order
    c.mu.Unlock()
}

func (c *ExCache) GetOrders(ids ...string) []*exsync.Order {
    c.mu.RLock()
    defer c.mu.RUnlock()
    orders := make([]*exsync.Order, 0, len(c.orders))
    if len(ids) > 0 {
        for _, id := range ids {
            if o := c.orders[id]; o != nil {
                orders = append(orders, o)
            }
        }
    } else {
        for _, o := range c.orders {
            orders = append(orders, o)
        }
    }
    return orders
}

func (c *ExCache) GetDoneOrders(ids ...string) []*exsync.Order {
    if len(ids) == 0 {
        return nil
    }
    all := c.GetOrders(ids...)
    orders := make([]*exsync.Order, 0, len(all))
    for _, o := range all {
        if IsOrderDone(o) {
            orders = append(orders, o)
        }
    }
    return orders
}

func (c *ExCache) GetOrder(id string) *exsync.Order {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.orders[id]
}

func IsOrderDone(order *exsync.Order) bool {
    if order == nil {
        return true
    }
    return order.GetStatus() == exsync.OrderStatus_Complete || order.GetStatus() == exsync.OrderStatus_Canceled
}

func (c *ExCache) SetDepth(asks, bids []*exsync.Trade) {
    c.mu.Lock()
    c.asks = asks
    c.bids = bids
    c.mu.Unlock()
}

func (c *ExCache) GetAsks() []*exsync.Trade {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.asks
}

func (c *ExCache) GetBids() []*exsync.Trade {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.bids
}

func (c *ExCache) SetIndex(i float64) {
    c.mu.Lock()
    c.index = i
    c.mu.Unlock()
}

func (c *ExCache) GetIndex() float64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.index
}

func (c *ExCache) NewTrade(trades []*exsync.Trade) {
    c.mu.Lock()
    if len(c.trades) > 21000 {
        c.trades = append(c.trades[1000:], trades...)
    } else {
        c.trades = append(c.trades, trades...)
    }
    c.mu.Unlock()
    c.saveTrades(trades)
    c.updateKline(trades)
}

func (c *ExCache) saveTrades(trades []*exsync.Trade) {
    for _, t := range trades {
        c.tradeLogger.Write("%s %s %f %.16f %.16f %d %d",
            t.GetId(), t.GetTAction(), t.GetAmount(), t.GetPrice(),
            t.GetFee(), t.GetCreateTime().GetSeconds(), t.GetCreateTime().GetNanos())
    }
}

func (c *ExCache) updateKline(trades []*exsync.Trade) {
    for _, t := range trades {
        if c.kline == nil {
            c.mu.Lock()
            c.kline = NewKline(c.Exname, t, time.Minute)
            c.klines = make([]*Kline, 0, 11000)
            c.mu.Unlock()
        } else {
            rt := c.kline.AddTrade(t)
            if rt == 1 {
                c.mu.Lock()
                if len(c.klines) > 11000 {
                    c.klines = append(c.klines[1000:], c.kline)
                } else {
                    c.klines = append(c.klines, c.kline)
                }
                c.kline = NewKline(c.Exname, t, time.Minute)
                c.mu.Unlock()
                //save
                var err error
                err = utils.Save(c.kline, "kline", utils.MainDB)
                if err != nil {
                    utils.FatalLog.Write(err.Error())
                }
                utils.DebugLog.Write("kline: %v", c.kline)
            }
        }
    }
}

func (c *ExCache) GetTrades(n int) []*exsync.Trade {
    c.mu.RLock()
    defer c.mu.RUnlock()
    if n < len(c.trades) {
        return c.trades[len(c.trades) - n:]
    }
    return c.trades
}

func (c *ExCache) GetKlines(n int) []*Kline {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.klines[len(c.klines) - n:]
}
