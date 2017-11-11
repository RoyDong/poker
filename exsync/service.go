package main

import (
    gctx "golang.org/x/net/context"
    "dw/poker/protobuf/exsync"
    "dw/poker/market/okex"
    "dw/poker/context"
    "dw/poker/market"
    "errors"
    "dw/poker/market/bitmex"
    "dw/poker/market/common"
)

type syncService struct {
    okexQuarter *okex.FutureSync
    okexWeek *okex.FutureSync
    bitmexXbtusd *bitmex.FutureSync
}

func newSyncService(conf *context.Config) (*syncService, error) {
    srv := &syncService{}

    //okex quarter
    var err error
    srv.okexQuarter, err = okex.NewFutureSync(
        conf.Market.Okex.ApiKey,
        conf.Market.Okex.ApiSecret,
        conf.Market.Okex.Wss,
        "quarter", market.OkexQuarter)
    if err != nil {
        return nil, err
    }
    srv.okexQuarter.AddHandler("NewTrade", func(args ...interface{}) {

    })

    //okex week
    srv.okexWeek, err = okex.NewFutureSync(
        conf.Market.OkexWeek.ApiKey,
        conf.Market.OkexWeek.ApiSecret,
        conf.Market.OkexWeek.Wss,
        "this_week", market.OkexWeek)
    if err != nil {
        return nil, err
    }

    //bitmex xbtusd
    srv.bitmexXbtusd, err = bitmex.NewFutureSync(
        conf.Market.Bitmex.ApiKey,
        conf.Market.Bitmex.ApiSecret,
        conf.Market.Bitmex.Wss, market.BitmexXbtusd)
    if err != nil {
        return nil, err
    }
    return srv, err
}

func (s *syncService) getCache(exname string) *common.ExCache {
    switch exname {
    case market.OkexQuarter:
        return &s.okexQuarter.ExCache

    case market.OkexWeek:
        return &s.okexWeek.ExCache

    case market.BitmexXbtusd:
        return &s.bitmexXbtusd.ExCache

    }
    return nil
}

func (s *syncService) Ping(ctx gctx.Context, in *exsync.ReqPing) (*exsync.Pong, error) {
    return &exsync.Pong{}, nil
}

func (s *syncService) GetOrders(ctx gctx.Context, in *exsync.ReqOrders) (*exsync.RespOrders, error) {
    cache := s.getCache(in.Exname)
    if cache == nil {
        return nil, errors.New("ex not found " + in.Exname)
    }
    resp := &exsync.RespOrders{}
    resp.Orders = cache.GetOrders(in.GetIds()...)
    return resp, nil
}

func (s *syncService) GetTrades(ctx gctx.Context, in *exsync.ReqTrades) (*exsync.RespTrades, error) {
    cache := s.getCache(in.Exname)
    if cache == nil {
        return nil, errors.New("ex not found " + in.Exname)
    }
    resp := &exsync.RespTrades{}
    resp.Trades = cache.GetTrades(int(in.Num))
    return resp, nil
}

func (s *syncService) GetDepth(ctx gctx.Context, in *exsync.Req) (*exsync.RespDepth, error) {
    cache := s.getCache(in.Exname)
    if cache == nil {
        return nil, errors.New("ex not found " + in.Exname)
    }
    resp := &exsync.RespDepth{}
    resp.Asks = cache.GetAsks()
    resp.Bids = cache.GetBids()
    return resp, nil
}

func (s *syncService) GetIndex(ctx gctx.Context, in *exsync.Req) (*exsync.RespIndex, error) {
    cache := s.getCache(in.Exname)
    if cache == nil {
        return nil, errors.New("ex not found " + in.Exname)
    }
    resp := &exsync.RespIndex{}
    resp.Index = cache.GetIndex()
    return resp, nil
}

func (s *syncService) GetPosition(ctx gctx.Context, in *exsync.Req) (*exsync.RespPosition, error) {
    cache := s.getCache(in.Exname)
    if cache == nil {
        return nil, errors.New("ex not found " + in.Exname)
    }
    resp := &exsync.RespPosition{}
    resp.Long = cache.GetLong()
    resp.Short = cache.GetShort()
    return resp, nil
}

func (s *syncService) GetBalance(ctx gctx.Context, in *exsync.Req) (*exsync.RespBalance, error) {
    cache := s.getCache(in.Exname)
    if cache == nil {
        return nil, errors.New("ex not found " + in.Exname)
    }
    resp := &exsync.RespBalance{}
    resp.Balance = cache.GetBalance()
    return resp, nil
}



