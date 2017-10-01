package market

import (
    "github.com/roydong/poker/common"
    "github.com/roydong/poker/market"
    "fmt"
    "strings"
    "github.com/roydong/poker/market/okex"
    "time"
)

type Market struct {

}


func (this *Market) Init(conf *common.Config) error {
    okconf := conf.Market.Okex
    ok := okex.NewFuture(okconf.HttpHost, okconf.ApiKey, okconf.ApiSecret)
    market.AddExchange(market.NewExchange(ok))
    return nil
}

func (this *Market) Run(ctx *common.Context) error {
    ok := market.GetExchange("okex/quarter")
    rows := make([]string, 0, 5)

    ticker, err := ok.GetTicker()
    i, err := ok.GetIndex()
    rows = append(rows, fmt.Sprintf("Price %.2f, Index %.2f, %.3f", ticker.Last, i, (ticker.Last - i) / i))

    b, err := ok.GetBalance()
    rows = append(rows, b.String())

    long, short, err := ok.GetPosition()
    rows = append(rows, long.String())
    rows = append(rows, short.String())
    rows = append(rows, "")
    ctx.RespBody = []byte(strings.Join(rows, "\n"))

    return err
}


