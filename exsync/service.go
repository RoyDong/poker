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
    "time"
)

type syncService struct {
    okexQuarter *okex.Future
    okexWeek *okex.Future
    bitmexXbtusd *bitmex.FutureSync
}

func newSyncService(conf *context.Config) (*syncService, error) {
    srv := &syncService{}

    //okex quarter
    var err error
    srv.okexQuarter = okex.NewFuture(
        conf.Market.Okex.HttpHost,
        conf.Market.Okex.Wss,
        conf.Market.Okex.ApiKey,
        conf.Market.Okex.ApiSecret,
        "quarter", market.OkexQuarter)

    err = srv.okexQuarter.StartSync()
    if err != nil {
        return nil, err
    }

    //okex week
    srv.okexWeek = okex.NewFuture(
        conf.Market.OkexWeek.HttpHost,
        conf.Market.OkexWeek.Wss,
        conf.Market.OkexWeek.ApiKey,
        conf.Market.OkexWeek.ApiSecret,
        "this_week", market.OkexWeek)

    err = srv.okexWeek.StartSync()
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
        return s.okexQuarter.ExCache

    case market.OkexWeek:
        return s.okexWeek.ExCache

    case market.BitmexXbtusd:
        //return s.bitmexXbtusd.ExCache

    }
    return nil
}

func (s *syncService) getExTrade(exname string) market.ITrade {
    switch exname {
    case market.OkexQuarter:
        return s.okexQuarter

    case market.OkexWeek:
        return s.okexWeek

    case market.BitmexXbtusd:
        //return &s.bitmexXbtusd.ExCache

    }
    return nil
}

func (s *syncService) Ping(ctx gctx.Context, in *exsync.ReqPing) (*exsync.Pong, error) {
    return &exsync.Pong{}, nil
}

func (s *syncService) MakeOrder(ctx gctx.Context, in *exsync.ReqMakeOrder) (*exsync.RespMakeOrder, error) {
    ex := s.getExTrade(in.GetExname())
    if ex == nil {
        return nil, errors.New("ex not found " + in.GetExname())
    }
    order, err := ex.MakeOrder(in.GetTAction(), in.GetAmount(), in.GetPrice())
    if err != nil {
        return nil, err
    }
    resp := &exsync.RespMakeOrder{}
    resp.Order = order
    return resp, nil
}

func (s *syncService) CancelOrders(ctx gctx.Context, in *exsync.ReqCancelOrder) (*exsync.Resp, error) {
    ex := s.getExTrade(in.GetExname())
    if ex == nil {
        return nil, errors.New("ex not found " + in.GetExname())
    }
    err := ex.CancelOrder(in.GetIds()...)
    if err != nil {
        return nil, err
    }
    resp := &exsync.Resp{}
    return resp, nil
}

func (s *syncService) GetOrders(ctx gctx.Context, in *exsync.ReqOrders) (*exsync.RespOrders, error) {
    cache := s.getCache(in.GetExname())
    if cache == nil {
        return nil, errors.New("ex not found " + in.GetExname())
    }
    resp := &exsync.RespOrders{}
    resp.Orders = cache.GetOrders(in.GetIds()...)
    return resp, nil
}

func (s *syncService) WaitOrders(ctx gctx.Context, in *exsync.ReqOrders) (*exsync.Resp, error) {
    cache := s.getCache(in.GetExname())
    resp := &exsync.Resp{}
    if cache == nil {
        return nil, errors.New("ex not found " + in.GetExname())
    }
    ids := in.GetIds()
    for i := 0; i < 100; i++ {
        tmpids := make([]string, 0, len(ids))
        for _, order := range cache.GetDoneOrders(ids...) {
            tmpids = append(tmpids, order.GetId())
        }
        if len(tmpids) == 0 {
            return resp, nil
        }
        ids = tmpids
        time.Sleep(10 * time.Millisecond)
    }
    return nil, errors.New("timeout")
}

func (s *syncService) GetTrades(ctx gctx.Context, in *exsync.ReqTrades) (*exsync.RespTrades, error) {
    cache := s.getCache(in.GetExname())
    if cache == nil {
        return nil, errors.New("ex not found " + in.GetExname())
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



