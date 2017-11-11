package common

import (
    "sync"
    "dw/poker/protobuf/exsync"
    "time"
    "dw/poker/utils"
)

type IExCache interface{


}

type ExCache struct {
    Exname string
    mu sync.RWMutex
    maxTradesLen int
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

func (c *ExCache) SetPosition(long, short *exsync.Position) {
    c.mu.Lock()
    c.long = long
    c.short = short
    c.mu.Unlock()
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

func (c *ExCache) NewTrade(t *exsync.Trade) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if len(c.trades) > c.maxTradesLen * 2 {
        c.trades = append(c.trades[c.maxTradesLen:], t)
    } else {
        c.trades = append(c.trades, t)
    }

    if c.kline == nil {
        c.kline = NewKline(c.Exname, t, time.Minute)
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
            c.kline = NewKline(c.Exname, t, time.Minute)
        }
    }

    tt := time.Unix(t.GetCreateTime().Seconds, t.GetCreateTime().Nanos)
    utils.DebugLog.Write("%s %s %v %v %v", c.Exname, t.TAction, t.Amount, t.Price, tt)
}

func (c *ExCache) GetTrades(n int) []*exsync.Trade {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.trades[len(c.trades) - n:]
}
