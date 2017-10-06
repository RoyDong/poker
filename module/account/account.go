package account

import (
    "dw/poker/context"
    "dw/poker/market"
    "dw/poker/utils"
)

type Account struct {

}


func (this *Account) Init(conf *context.Config) error {

    return nil
}

func (this *Account) Run(ctx *context.Context) error {

    ok := market.GetExchange("okex/quarter")

    p := ok.LastnAvgPrice(10)

    utils.DebugLog.Write("last 5 avg price %f", p)
    return nil
}
