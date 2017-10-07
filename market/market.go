package market

import (
    pctx "dw/poker/context"
    "dw/poker/market/okex"
)


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


