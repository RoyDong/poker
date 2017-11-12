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
    tradePipe chan *exsync.Trade

    index float64

    asks []*exsync.Trade
    bids []*exsync.Trade

    orders map[string]*exsync.Order

    long *exsync.Position
    short *exsync.Position

    balance *exsync.Balance

    kline *Kline
    klines []*Kline
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
    if c.long == nil {
        c.long = &exsync.Position{}
    }
    if c.short == nil {
        c.short = &exsync.Position{}
    }
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
    if c.orders == nil {
        c.orders = make(map[string]*exsync.Order, 10)
    }
    c.orders[order.GetId()] = order
    c.mu.Unlock()
}

func (c *ExCache) GetOrders(ids ...string) []*exsync.Order {
    c.mu.RLock()
    defer c.mu.RUnlock()
    orders := make([]*exsync.Order, 0, 10)
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

func (c *ExCache) GetOrder(id string) *exsync.Order {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.orders[id]
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
    if len(c.trades) > 2000 {
        c.trades = append(c.trades[200:], trades...)
    } else {
        c.trades = append(c.trades, trades...)
    }
    c.mu.Unlock()
}

func (c *ExCache) UpdateKline(trades []*exsync.Trade) {
    for _, t := range trades {
        if c.kline == nil {
            c.mu.Lock()
            c.kline = NewKline(c.Exname, t, time.Minute)
            c.klines = make([]*Kline, 0, 10000)
            c.mu.Unlock()
        } else {
            rt := c.kline.AddTrade(t)
            if rt == 1 {
                //save
                var err error
                err = utils.Save(c.kline, "kline", utils.MainDB)
                if err != nil {
                    utils.FatalLog.Write(err.Error())
                }
                utils.DebugLog.Write("kline: %v", c.kline)
                c.mu.Lock()
                if len(c.klines) > 10000 {
                    c.klines = append(c.klines[2000:], c.kline)
                } else {
                    c.klines = append(c.klines, c.kline)
                }
                c.kline = NewKline(c.Exname, t, time.Minute)
                c.mu.Unlock()
            }
        }
        tt := time.Unix(t.GetCreateTime().Seconds, t.GetCreateTime().Nanos)
        utils.DebugLog.Write("%s %s %v %v %v", c.Exname, t.TAction, t.Amount, t.Price, tt)
    }
}

func (c *ExCache) GetTrades(n int) []*exsync.Trade {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.trades[len(c.trades) - n:]
}

func (c *ExCache) GetKlines(n int) []*Kline {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.klines[len(c.klines) - n:]
}
