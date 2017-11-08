package main

import (
    gctx "golang.org/x/net/context"
    "dw/poker/proto/exsync"
    "dw/poker/market/okex"
    "dw/poker/context"
    mctx "dw/poker/market/context"
    "sync"
    "dw/poker/utils"
    "dw/poker/market"
    "errors"
    "time"
    "dw/poker/market/bitmex"
)

type dataCache struct {
    exname string
    syncEvent utils.IEvent

    mu sync.RWMutex
    maxTradesLen int
    trades []*exsync.Trade

    index float64

    long *exsync.Position
    short *exsync.Position

    balance *exsync.Balance
}


func (c *dataCache) init() {
    c.maxTradesLen = 5
    var kline *mctx.Kline
    c.syncEvent.AddHandler("NewTrade", func(args ...interface{}) {
        if len(args) != 1 {
            return
        }
        trades, ok := args[0].([]*exsync.Trade)
        if !ok {
            return
        }
        overflow := len(c.trades) + len(trades) - c.maxTradesLen
        if overflow < 0 {
            overflow = 0
        } else if overflow > len(c.trades) {
            overflow = len(c.trades)
        }
        c.mu.Lock()
        c.trades = append(c.trades[overflow:], trades...)
        c.mu.Unlock()

        for _, t := range trades {
            tt := time.Unix(t.GetCreateTime().Seconds, t.GetCreateTime().Nanos)
            utils.DebugLog.Write("%s %s %v %v %v", c.exname, t.TAction, t.Amount, t.Price, tt)
        }

        //create kline and save to sql db
        for _, t := range trades {
            if kline == nil {
                kline = mctx.NewKline(c.exname, t, time.Minute)
            } else {
                rt := kline.AddTrade(t)
                if rt == 1 {
                    //save
                    var err error
                    //err = utils.Save(kline, "kline", nil)
                    if err != nil {
                        utils.FatalLog.Write(err.Error())
                    }
                    utils.DebugLog.Write("kline: %v", kline)
                    kline = mctx.NewKline(c.exname, t, time.Minute)
                }
            }
        }
    })

    c.syncEvent.AddHandler("IndexUpdate", func(args ...interface{}) {
        if len(args) != 1 {
            return
        }
        index, ok := args[0].(float64)
        if !ok {
            return
        }
        c.mu.Lock()
        c.index = index
        c.mu.Unlock()
        utils.DebugLog.Write(c.exname + " index update %f", index)
    })

    /*
    c.syncEvent.AddHandler("position_update")

    c.syncEvent.AddHandler("balance_update")
    */
}


type syncService struct {
    okexQuarter *dataCache
    okexWeek *dataCache
    bitmexXbtusd *dataCache
}

func newSyncService(conf *context.Config) (*syncService, error) {
    srv := &syncService{}

    //okex quarter
    var err error
    cache := &dataCache{}
    cache.exname = market.OkexQuarter
    cache.syncEvent, err = okex.NewFutureSync(
        conf.Market.Okex.ApiKey,
        conf.Market.Okex.ApiSecret,
        conf.Market.Okex.Wss,
        "quarter")
    if err != nil {
        return nil, err
    }
    cache.init()
    srv.okexQuarter = cache

    //okex week
    cache = &dataCache{}
    cache.exname = market.OkexWeek
    cache.syncEvent, err = okex.NewFutureSync(
        conf.Market.Okex.ApiKey,
        conf.Market.Okex.ApiSecret,
        conf.Market.Okex.Wss,
        "this_week")
    if err != nil {
        return nil, err
    }
    cache.init()
    srv.okexWeek = cache

    //bitmex xbtusd
    cache = &dataCache{}
    cache.exname = market.BitmexXbtusd
    cache.syncEvent, err = bitmex.NewFutureSync(
        conf.Market.Bitmex.ApiKey,
        conf.Market.Bitmex.ApiSecret,
        conf.Market.Bitmex.Wss)
    if err != nil {
        return nil, err
    }
    cache.init()
    srv.bitmexXbtusd = cache

    return srv, err
}

func (s *syncService) getCache(exname string) *dataCache {
    switch exname {
    case market.OkexQuarter:
        return s.okexQuarter

    case market.OkexWeek:
        return s.okexWeek

    case market.BitmexXbtusd:
        return s.bitmexXbtusd

    }
    return nil
}

func (s *syncService) Ping(ctx gctx.Context, in *exsync.ReqPing) (*exsync.Pong, error) {
    return &exsync.Pong{}, nil
}

func (s *syncService) GetOrder(ctx gctx.Context, in *exsync.ReqOrder) (*exsync.RespOrder, error) {
    return nil, nil
}

func (s *syncService) GetTrades(ctx gctx.Context, in *exsync.ReqTrades) (*exsync.RespTrades, error) {
    cache := s.getCache(in.Exname)
    if cache == nil {
        return nil, errors.New("ex not found " + in.Exname)
    }
    resp := &exsync.RespTrades{}
    cache.mu.RLock()
    resp.Trades = cache.trades[-in.Num:]
    cache.mu.RUnlock()
    return resp, nil
}

func (s *syncService) GetDepth(ctx gctx.Context, in *exsync.Req) (*exsync.RespDepth, error) {

    return nil, nil
}

func (s *syncService) GetIndex(ctx gctx.Context, in *exsync.Req) (*exsync.RespIndex, error) {
    cache := s.getCache(in.Exname)
    if cache == nil {
        return nil, errors.New("ex not found " + in.Exname)
    }
    resp := &exsync.RespIndex{}
    cache.mu.RLock()
    resp.Index = cache.index
    cache.mu.RUnlock()
    return resp, nil
}

func (s *syncService) GetPosition(ctx gctx.Context, in *exsync.Req) (*exsync.RespPosition, error) {

    return nil, nil
}

func (s *syncService) GetBalance(ctx gctx.Context, in *exsync.Req) (*exsync.RespBalance, error) {

    return nil, nil
}



