package market

import (
    "sync"
    "time"
    "dw/poker/utils"
    "dw/poker/market/context"
    "math"
    "errors"
    "dw/poker/proto"
    "log"
    "google.golang.org/grpc"
    gctx "golang.org/x/net/context"
)

var RPCTimeout = 200 * time.Millisecond

type ITrade interface {
    MakeOrder(ta context.TradeAction, amount, price float64) (context.Order, error)
    CancelOrder(id ...string) error
}

type Exchange struct {
    ITrade
    *utils.Event

    exsyncHost string
    exsyncMu sync.RWMutex
    exsync exsync.SyncServiceClient

    name string
    tradeMu sync.RWMutex
    inLoop bool
    maxTradesLen int
    trades []*exsync.Trade
    maxKlinesLen int
    klines []*context.Kline
}

func NewExchange(name, exsyncHost string) *Exchange {
    ex := &Exchange{}
    ex.Event = utils.NewEvent()
    ex.name = name
    ex.maxTradesLen = 10000
    ex.trades = make([]*exsync.Trade, 1, ex.maxTradesLen)
    ex.maxKlinesLen = 100
    ex.klines = make([]*context.Kline, 0, ex.maxKlinesLen)
    ex.inLoop = true

    ex.exsync = ex.createExsyncClient()
    go ex.keepExsyncClinet()
    go ex.syncTrades()
    return ex
}

func (ex *Exchange) Name() string {
    return ex.name
}

func (ex *Exchange) keepExsyncClinet() {
    for {
        time.Sleep(2 * time.Minute)
        _, err := ex.getExsyncClient().Ping(ex.timeoutCtx(), &exsync.ReqPing{})
        if err != nil {
            ex.exsyncMu.Lock()
            ex.exsync = ex.createExsyncClient()
            ex.exsyncMu.Unlock()
        }
    }
}

func (ex *Exchange) createExsyncClient() exsync.SyncServiceClient {
    conn, err := grpc.Dial(ex.exsyncHost, grpc.WithInsecure())
    if err != nil {
        log.Fatal("did not connect: %v", err)
    }
    return exsync.NewSyncServiceClient(conn)
}

func (ex *Exchange) getExsyncClient() exsync.SyncServiceClient {
    ex.exsyncMu.RLock()
    defer ex.exsyncMu.RUnlock()
    return ex.exsync
}

func (ex *Exchange) timeoutCtx() gctx.Context {
    ctx, _ := gctx.WithTimeout(gctx.Background(), RPCTimeout)
    return ctx
}

func (ex *Exchange) GetTrades() ([]*exsync.Trade, error) {
    resp, err := ex.exsync.GetTrades(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Trades, err
}

func (ex *Exchange) GetOrder(id string) (*exsync.Order, error) {
    resp, err := ex.exsync.GetOrder(ex.timeoutCtx(), &exsync.ReqOrder{Exname:ex.name, Id:id})
    return resp.Order, err
}

func (ex *Exchange) GetDepth() ([]*exsync.Order, []*exsync.Order, error) {
    resp, err := ex.exsync.GetDepth(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Asks, resp.Bids, err
}

func (ex *Exchange) GetIndex() (float64, error) {
    resp, err := ex.exsync.GetIndex(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Index, err
}

func (ex *Exchange) GetPosition() (*exsync.Position, *exsync.Position, error) {
    resp, err := ex.exsync.GetPosition(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Long, resp.Short, err
}

func (ex *Exchange) GetBalance() (*exsync.Balance, error) {
    resp, err := ex.exsync.GetBalance(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Balance, err
}

func (ex *Exchange) syncTrades() {
    var kline *context.Kline
    for ex.inLoop {
        time.Sleep(200 * time.Millisecond)
        trades, err := ex.GetTrades()
        if err != nil {
            utils.WarningLog.Write("sync trade err: %v", err.Error())
            continue
        }
        newTrades := make([]*exsync.Trade, 0, len(trades))
        for _, trade := range trades {
            for i := len(ex.trades) - 1; i >= 0; i-- {
                t := ex.trades[i]
                delta := trade.GetCreateTime().Seconds - t.GetCreateTime().Seconds
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
            overflow := len(ex.trades) + len(newTrades) - ex.maxTradesLen
            if overflow < 0 {
                overflow = 0
            } else if overflow > len(ex.trades) {
                overflow = len(ex.trades)
            }
            ex.tradeMu.Lock()
            ex.trades = append(ex.trades[overflow:], newTrades...)
            ex.tradeMu.Unlock()

            //create kline and save to sql db
            for _, t := range newTrades {
                if kline == nil {
                    kline = context.NewKline(ex.Name(), t, time.Minute)
                } else {
                    rt := kline.AddTrade(t)
                    if rt == 1 {
                        //save
                        ex.Trigger("kline_close", kline)
                        err := utils.Save(kline, "kline", utils.MainDB)
                        if err != nil {
                            utils.FatalLog.Write(err.Error())
                        }
                        kline = context.NewKline(ex.Name(), t, time.Minute)

                        offset := 0
                        if len(ex.klines) >= ex.maxKlinesLen {
                            offset = 1
                        }
                        ex.tradeMu.Lock()
                        ex.klines = append(ex.klines[offset:], kline)
                        ex.tradeMu.Unlock()
                    }
                }
            }
        }
    }
}

func (ex *Exchange) LastTrades() []*exsync.Trade {
    ex.tradeMu.RLock()
    defer ex.tradeMu.RUnlock()
    return ex.trades
}

func (ex *Exchange) LastKlines() []*context.Kline {
    ex.tradeMu.RLock()
    defer ex.tradeMu.RUnlock()
    return ex.klines
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

func (ex *Exchange) OrderCompleteOrPriceChange(order *exsync.Order, spread float64) (*exsync.Order, bool) {
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
    var order *exsync.Order
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

