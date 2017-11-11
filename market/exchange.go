package market

import (
    "sync"
    "time"
    "dw/poker/utils"
    "math"
    "errors"
    "google.golang.org/grpc"
    gctx "golang.org/x/net/context"
    "dw/poker/protobuf/exsync"
    "dw/poker/market/common"
)

var RPCTimeout = 10 * time.Millisecond

type ITrade interface {
    MakeOrder(ta exsync.TradeAction, amount, price float64) (*exsync.Order, error)
    CancelOrder(id ...string) error
}

type Exchange struct {
    ITrade
    utils.Event

    exsyncHost string
    exsyncMu sync.RWMutex
    exsync exsync.SyncServiceClient

    name string
    tradeMu sync.RWMutex
    inLoop bool
    trades []*exsync.Trade
    maxKlinesLen int
    klines []*common.Kline
}

func NewExchange(name, exsyncHost string, itrade ITrade) *Exchange {
    ex := &Exchange{}
    ex.name = name
    ex.ITrade = itrade
    ex.maxKlinesLen = 100
    ex.klines = make([]*common.Kline, 0, ex.maxKlinesLen)
    ex.inLoop = true

    var err error
    ex.exsyncHost = exsyncHost
    ex.exsync, err = ex.createExsyncClient()
    if err != nil {
        return nil
    }
    return ex
}

func (ex *Exchange) Name() string {
    return ex.name
}

func (ex *Exchange) createExsyncClient() (exsync.SyncServiceClient, error) {
    conn, err := grpc.Dial(ex.exsyncHost, grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    return exsync.NewSyncServiceClient(conn), nil
}

func (ex *Exchange) getExsyncClient() exsync.SyncServiceClient {
    ex.exsyncMu.RLock()
    defer ex.exsyncMu.RUnlock()
    _, err := ex.exsync.Ping(ex.timeoutCtx(), &exsync.ReqPing{})
    if err != nil {
        ex.exsyncMu.Lock()
        ex.exsync, _ = ex.createExsyncClient()
        ex.exsyncMu.Unlock()
    }
    return ex.exsync
}

func (ex *Exchange) timeoutCtx() gctx.Context {
    ctx, _ := gctx.WithTimeout(gctx.Background(), RPCTimeout)
    return ctx
}

func (ex *Exchange) GetTrades(n int32) ([]*exsync.Trade, error) {
    resp, err := ex.getExsyncClient().GetTrades(ex.timeoutCtx(), &exsync.ReqTrades{Exname:ex.name, Num:n})
    if err != nil {
        utils.FatalLog.Write(err.Error())
        return nil, err
    }
    return resp.Trades, err
}

func (ex *Exchange) GetOrder(id string) (*exsync.Order, error) {
    resp, err := ex.getExsyncClient().GetOrders(ex.timeoutCtx(), &exsync.ReqOrders{Exname:ex.name, Ids:[]string{id}})
    if err != nil {
        return nil, err
    }
    if len(resp.GetOrders()) == 1 {
        return resp.GetOrders()[0], nil
    }
    return nil, err
}

func (ex *Exchange) GetOrders() ([]*exsync.Order, error) {
    resp, err := ex.getExsyncClient().GetOrders(ex.timeoutCtx(), &exsync.ReqOrders{Exname:ex.name})
    if err != nil {
        return nil, err
    }
    return resp.GetOrders(), nil
}

func (ex *Exchange) GetDepth() ([]*exsync.Trade, []*exsync.Trade, error) {
    resp, err := ex.getExsyncClient().GetDepth(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Asks, resp.Bids, err
}

func (ex *Exchange) GetIndex() (float64, error) {
    resp, err := ex.getExsyncClient().GetIndex(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Index, err
}

func (ex *Exchange) GetPosition() (*exsync.Position, *exsync.Position, error) {
    resp, err := ex.getExsyncClient().GetPosition(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Long, resp.Short, err
}

func (ex *Exchange) GetBalance() (*exsync.Balance, error) {
    resp, err := ex.getExsyncClient().GetBalance(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    return resp.Balance, err
}

func (ex *Exchange) LastTrades() []*exsync.Trade {
    ex.tradeMu.RLock()
    defer ex.tradeMu.RUnlock()
    return ex.trades
}

func (ex *Exchange) LastKlines() []*common.Kline {
    ex.tradeMu.RLock()
    defer ex.tradeMu.RUnlock()
    return ex.klines
}

func (ex *Exchange) LastnAvgPrice(n int32) float64 {
    trades, err := ex.GetTrades(n)
    if err != nil {
        return 0
    }
    var sum float64
    for _, t := range trades {
        sum += t.Price
    }
    return sum / float64(n)
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
func (ex *Exchange) TakeDepth(ta exsync.TradeAction, amount float64) (*exsync.Order, error) {
    var price float64
    var err error
    var order *exsync.Order
    switch ta {
    case exsync.TradeAction_OpenLong, exsync.TradeAction_CloseShort, exsync.TradeAction_Buy:
        price, err = ex.GetAskPrice(amount)
    case exsync.TradeAction_OpenShort, exsync.TradeAction_CloseLong, exsync.TradeAction_Sell:
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
            if o.Status == exsync.OrderStatus_OrderStatusComplete ||
                o.Status == exsync.OrderStatus_OrderStatusCanceled {
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

func (ex *Exchange) Trade(ta exsync.TradeAction, amount, price, spread float64) (*exsync.Order, error) {
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
    Balance *exsync.Balance
    Long *exsync.Position
    Short *exsync.Position
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

