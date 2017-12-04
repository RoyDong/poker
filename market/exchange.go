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
    "fmt"
    "bufio"
    "os"
    "strings"
    "strconv"
)

var RPCTimeout = 10 * time.Millisecond

type ITrade interface {
    MakeOrder(ta exsync.TradeAction, amount, price float64) (*exsync.Order, error)
    CancelOrder(ids ...string) error
}

type Exchange struct {
    utils.Event
    name string

    exsyncHost string
    exsyncMu sync.RWMutex
    exsync exsync.SyncServiceClient

    mu sync.RWMutex
    klines []*common.Kline
}

func NewExchange(name, exsyncHost string) *Exchange {
    ex := &Exchange{}
    ex.name = name
    var err error
    ex.exsyncHost = exsyncHost
    ex.exsync, err = ex.createExsyncClient()
    if err != nil {
        return nil
    }
    go ex.loadKlines()
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

func (ex *Exchange) MakeOrder(ta exsync.TradeAction, amount, price float64) (*exsync.Order, error) {
    req := &exsync.ReqMakeOrder{
        Exname:ex.name,
        TAction:ta,
        Amount:amount,
        Price:price,
    }
    resp, err := ex.getExsyncClient().MakeOrder(gctx.Background(), req)
    if err != nil {
        return nil, err
    }
    return resp.Order, nil
}

func (ex *Exchange) CancelOrder(ids ...string) error {
    _, err := ex.getExsyncClient().CancelOrder(gctx.Background(), &exsync.ReqCancelOrder{Exname:ex.name,Ids:ids})
    return err
}

func (ex *Exchange) CancelAllOrders() error {
    return ex.CancelOrder("-1")
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
    return nil, errors.New("order not found " + id)
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
    if err != nil {
        return nil, nil, err
    }
    return resp.Asks, resp.Bids, err
}

func (ex *Exchange) GetIndex() (float64, error) {
    resp, err := ex.getExsyncClient().GetIndex(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    if err != nil {
        return 0, err
    }
    return resp.Index, err
}

func (ex *Exchange) GetPosition() (*exsync.Position, *exsync.Position, error) {
    resp, err := ex.getExsyncClient().GetPosition(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    if err != nil {
        return nil, nil, err
    }
    return resp.Long, resp.Short, err
}

func (ex *Exchange) GetBalance() (*exsync.Balance, error) {
    resp, err := ex.getExsyncClient().GetBalance(ex.timeoutCtx(), &exsync.Req{Exname:ex.name})
    if err != nil {
        return nil, err
    }
    return resp.Balance, err
}

func (ex *Exchange) LastnAvgPrice(n int32) float64 {
    trades, err := ex.GetTrades(n)
    if err != nil {
        return 0
    }
    return GetAvgPrice(trades)
}

func (ex *Exchange) GetKlines(n int) []*common.Kline {
    ex.mu.RLock()
    defer ex.mu.RUnlock()
    return ex.klines[len(ex.klines) - n:]
}

func (ex *Exchange) loadKlines() {
    lastid := 0
    for {
        stmt := "select * from kline where exname = ? and id > ? order by open_time desc limit 10"
        r, err := utils.MainDB.Query(stmt, ex.name, lastid)
        if err != nil {
            utils.WarningLog.Write("load kline from db error %s", err.Error())
            return
        }
        var klines []*common.Kline
        for r.Next() {
            var k *common.Kline
            err = utils.Scan(r, &k)
            if err != nil {
                utils.WarningLog.Write("scan kline from db error %s", err.Error())
                continue
            }
            klines = append(klines, k)
        }
        for i := len(klines) - 1; i >= 0; i-- {
            v := klines[i]
            ex.mu.Lock()
            ex.klines = append(ex.klines, v)
            ex.mu.Unlock()
            ex.Trigger("KlineClose", v)
            utils.DebugLog.Write(ex.name + " load kline from db %v", v)
            lastid = v.Id
        }
        time.Sleep(5 * time.Second)
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
            if o.Status == exsync.OrderStatus_Complete ||
                o.Status == exsync.OrderStatus_Canceled {
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
        time.Sleep(500 * time.Millisecond)
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

func (ex *Exchange) LoadCandles(days int, s int64) []*common.Candle {
    day, err := time.ParseDuration("24h")
    if err != nil {
        utils.FatalLog.Write("get 24h time duration error %s", err.Error())
    }
    now := time.Now()
    candles := make([]*common.Candle, 0, 10000)
    var candle *common.Candle
    for d := days; d >= 0; d-- {
        var logfile string
        if d == 0 {
            logfile = fmt.Sprintf("/opt/poker/exdata/%s-trade.log", ex.Name())
        } else {
            date := now.Add(-time.Duration(d) * day).Format("20060102")
            logfile = fmt.Sprintf("/opt/poker/exdata/%s-trade.log-%s", ex.Name(), date)
        }
        fp, err := os.Open(logfile)
        if err != nil {
            utils.WarningLog.Write("read trade log error %s", err.Error())
            continue
        }
        scanner := bufio.NewScanner(fp)
        for scanner.Scan() {
            row := scanner.Text()
            arr := strings.Split(row, " ")
            if len(arr) == 9 {
                trade := &exsync.Trade{}
                trade.Id = arr[2]
                trade.TAction = exsync.TradeAction(exsync.TradeAction_value[arr[3]])
                trade.Amount, _ = strconv.ParseFloat(arr[4], 64)
                trade.Price, _ = strconv.ParseFloat(arr[5], 64)
                trade.Fee, _ = strconv.ParseFloat(arr[6], 64)
                sec, _ := strconv.ParseInt(arr[7], 10, 64)
                nanos, _ := strconv.ParseInt(arr[8], 10, 64)
                trade.CreateTime = &exsync.Timestamp{sec, nanos}
                if trade.Amount > 0 && trade.Price > 0 {
                    if candle == nil {
                        candle = common.NewCandle(ex.Name(), trade, s)
                    } else if candle.AddTrade(trade) == 1 {
                        if candle.Amount > 0 && candle.AvgPrice > 0 {
                            candles = append(candles, candle)
                        }
                        candle = common.NewCandle(ex.Name(), trade, s)
                    }
                }
            }
        }
        fp.Close()

    }
    return candles
}


