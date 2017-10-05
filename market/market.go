package market

import (
    "time"
    "github.com/roydong/poker/context"
    "github.com/roydong/poker/market/okex"
    "github.com/roydong/poker/market/base"
)
type IExchange interface {
    Name() string

    GetCurrencyUnit() base.CurrencyUnit

    OpenTime() time.Time

    CloseTime() time.Time

    /*
    pos    开仓类型
    amount 数量
    price 价格  price = 0 市价, price = -1 对手价
     */
    MakeOrder(ta base.TradeAction, amount, price float64) (base.Order, error)

    CancelOrder(id ...string) error

    GetOrder(id string) (base.Order, error)

    GetOrders(ids []string) ([]base.Order, error)

    GetTicker() (base.Ticker, error)

    GetTrades() ([]base.Trade, error)

    GetDepth() ([]base.Order, []base.Order, error)

    GetIndex() (float64, error)

    GetBalance() (base.Balance, error)

    GetPosition() (base.Position, base.Position, error)
}

var exchanges = make(map[string]*Exchange, 0)

func AddExchange(ex *Exchange) {
    exchanges[ex.Name()] = ex
}

func GetExchange(name string) *Exchange {
    return exchanges[name]
}

func Init(conf *context.Config) {
    okconf := conf.Market.Okex
    ok := okex.NewFuture(okconf.HttpHost, okconf.ApiKey, okconf.ApiSecret)
    AddExchange(NewExchange(ok))
}


