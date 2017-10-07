package account

import (
    "dw/poker/context"
    "dw/poker/market"
    context2 "dw/poker/market/context"
)

type Account struct {

    inLoop bool
}


func (this *Account) Init(conf *context.Config) error {
    return nil
}

func (this *Account) Run(ctx *context.Context) error {
    return nil
}


