package market

import (
    pctx "dw/poker/context"
    "dw/poker/market/okex"
    "dw/poker/market/context"
)

type IExchange interface {
    Name() string

    GetCurrencyUnit() context.CurrencyUnit

    /*
    pos    开仓类型
    amount 数量
    price 价格  price = 0 市价, price = -1 对手价
     */
    MakeOrder(ta context.TradeAction, amount, price float64) (context.Order, error)

    CancelOrder(id ...string) error

    GetOrder(id string) (context.Order, error)

    GetOrders(ids []string) ([]context.Order, error)

    GetTicker() (context.Ticker, error)

    GetTrades() ([]context.Trade, error)

    GetDepth() ([]context.Order, []context.Order, error)

    GetIndex() (float64, error)

    GetBalance() (context.Balance, error)

    GetPosition() (context.Position, context.Position, error)
}

var exchanges = make(map[string]*Exchange, 0)

func AddExchange(ex *Exchange) {
    exchanges[ex.Name()] = ex
}

func GetExchange(name string) *Exchange {
    return exchanges[name]
}

func Init(conf *pctx.Config) {
    okconf := conf.Market.Okex
    ok := okex.NewFuture(okconf.HttpHost, okconf.ApiKey, okconf.ApiSecret)
    AddExchange(NewExchange(ok))
}


