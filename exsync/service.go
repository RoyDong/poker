package main

import (
    gctx "golang.org/x/net/context"
    "dw/poker/proto/exsync"
    "dw/poker/market/okex"
    "dw/poker/context"
    mctx "dw/poker/market/context"
    "sync"
    "dw/poker/utils"
)

type dataCache struct {
    syncEvent utils.IEvent

    tradesMu sync.RWMutex
    trades []*exsync.Trade
    klines []*mctx.Kline

    indexMu sync.RWMutex
    index float64

    positionMu sync.RWMutex
    long *exsync.Position
    short *exsync.Position

    balanceMu sync.RWMutex
    balance *exsync.Balance
}


func (c *dataCache) init() {
    /*
    c.syncEvent.AddHandler("new_trade")

    c.syncEvent.AddHandler("index_update")

    c.syncEvent.AddHandler("position_update")

    c.syncEvent.AddHandler("balance_update")
    */
}


type syncService struct {
    okexQuarter *dataCache
}

func newSyncService(conf *context.Config) (*syncService, error) {
    sync := &syncService{}

    var err error
    cache := &dataCache{}
    cache.syncEvent, err = okex.NewFutuerSync(
        conf.Market.Okex.ApiKey,
        conf.Market.Okex.ApiSecret,
        conf.Market.Okex.Wss,
        "quarter")

    if err != nil {
        return nil, err
    }

    sync.okexQuarter = cache

    return sync, err
}

func (s *syncService) Ping(ctx gctx.Context, in *exsync.ReqPing) (*exsync.Pong, error) {
    return &exsync.Pong{}, nil
}

func (s *syncService) GetOrder(ctx gctx.Context, in *exsync.ReqOrder) (*exsync.RespOrder, error) {
    return nil, nil
}

func (s *syncService) GetTrades(ctx gctx.Context, in *exsync.ReqTrades) (*exsync.RespTrades, error) {

    return nil, nil
}

func (s *syncService) GetDepth(ctx gctx.Context, in *exsync.Req) (*exsync.RespDepth, error) {

    return nil, nil
}

func (s *syncService) GetIndex(ctx gctx.Context, in *exsync.Req) (*exsync.RespIndex, error) {

    return nil, nil
}

func (s *syncService) GetPosition(ctx gctx.Context, in *exsync.Req) (*exsync.RespPosition, error) {

    return nil, nil
}

func (s *syncService) GetBalance(ctx gctx.Context, in *exsync.Req) (*exsync.RespBalance, error) {

    return nil, nil
}



