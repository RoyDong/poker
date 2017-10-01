package riskctrl

import (
    "github.com/roydong/poker/common"
    "github.com/roydong/poker/market"
    "errors"
    "github.com/roydong/poker/utils"
    "time"
    "sync"
    "fmt"
    "github.com/roydong/poker/market/okex"
    "strings"
)

type RiskCtrl struct {

}


func (this *RiskCtrl) Init(conf *common.Config) error {
    return nil
}


func (this *RiskCtrl) Run(ctx *common.Context) error {
    go baseCtrl()
    ctx.RespBody = []byte("base control n")
    return nil
}


func baseCtrl() error {
    ok := market.GetExchange("okex/quarter")
    if ok == nil {
        return errors.New("Risk control error exchange not found " + ok.Name())
    }
    n := 0
    for {
        <- time.After(5 * time.Second)
        wg := sync.WaitGroup{}
        wg.Add(4)
        var balance market.Balance
        var long, short market.Position
        var ticker market.Ticker
        var index float64
        go func() {
            balance, _ = ok.GetBalance()
            wg.Done()
        }()
        go func() {
            long, short, _ = ok.GetPosition()
            wg.Done()
        }()
        go func() {
            ticker, _ = ok.GetTicker()
            wg.Done()
        }()
        go func() {
            index, _ = ok.GetIndex()
            wg.Done()
        }()
        wg.Wait()

        lprofit := long.GetProfit(ticker.Last)
        lrop := long.GetROP(ticker.Last)
        sprofit := short.GetProfit(ticker.Last)
        srop := short.GetROP(ticker.Last)

        msg := make([]string, 0, 2)
        msg = append(msg, fmt.Sprintf("Price %.4f Index %.4f", okex.FutureBTC_USD(ticker.Last), okex.FutureBTC_USD(index)))
        if lrop < -0.20 {
            ok.TakeDepth(market.CloseLong, long.AvailableAmount)
            tpl := `空头(usd long) %.4f/%.4f Deposit %.4f Profit %.4f`
            msg = append(msg, fmt.Sprintf(tpl, long.Amount, long.AvailableAmount, long.Deposit, lprofit))
        }
        if srop < -0.20 {
            ok.TakeDepth(market.CloseShort, short.AvailableAmount)
            tpl := `多头(usd short) %.4f/%.4f Deposit %.4f Profit %.4f`
            msg = append(msg, fmt.Sprintf(tpl, short.Amount, short.AvailableAmount, short.Deposit, sprofit))
        }

        if len(msg) > 1 || n > 600 {
            utils.SendSysMail(strings.Join(msg, "\n===========\n"), fmt.Sprintf("Loss long=%.3f short=%.3f", srop, lrop))
            n = 0
        }
        n++
        utils.NoticeLog.Write("Loss long=%.3f short=%.3f", srop, lrop)
    }
}