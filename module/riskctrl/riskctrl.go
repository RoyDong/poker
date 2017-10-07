package riskctrl

import (
    "time"
    "sync"
    "fmt"
    "strings"
    "dw/poker/utils"
    "dw/poker/context"
    "dw/poker/market"
    "dw/poker/market/okex"
    mctx "dw/poker/market/context"
)

type RiskCtrl struct {

    inLoop bool
}


func (this *RiskCtrl) Init(conf *context.Config) error {
    go this.baseCtrl()
    return nil
}


func (this *RiskCtrl) Run(ctx *context.Context) error {
    ctx.RespBody = []byte("base control n")
    return nil
}


func (this *RiskCtrl) baseCtrl() {
    if this.inLoop {
        return
    }
    ok := market.GetExchange("okex/quarter")
    if ok == nil {
        utils.FatalLog.Write("okex exchange not found")
        return
    }
    n := 0
    this.inLoop = true
    for this.inLoop {
        <- time.After(10 * time.Second)
        wg := sync.WaitGroup{}
        wg.Add(3)
        var balance mctx.Balance
        var long, short mctx.Position
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
            index, _ = ok.GetIndex()
            wg.Done()
        }()
        wg.Wait()

        price := ok.LastnAvgPrice(5)
        lprofit := long.GetProfit(price)
        lrop := long.GetROP(price)
        sprofit := short.GetProfit(price)
        srop := short.GetROP(price)

        usdprice := okex.FutureBTC_USD(price)
        usdindex := okex.FutureBTC_USD(index)

        rows := make([]string, 0, 3)
        rows = append(rows, fmt.Sprintf("P[%.2f %.1f%%]", usdprice, (usdprice - usdindex) / usdindex * 100))

        msg := make([]string, 0, 2)
        hasPosition := false
        if long.Amount > 0 {
            tpl := `空头(usd long) %.4f/%.4f Deposit %.4f Profit %.4f`
            msg = append(msg, fmt.Sprintf(tpl, long.Amount, long.AvailableAmount, long.Deposit, lprofit))
            rows = append(rows, fmt.Sprintf("S[%.0f %.4f %.1f%%]", long.Amount, lprofit, lrop * 100))
            hasPosition = true
        }
        if short.Amount > 0 {
            tpl := `多头(usd short) %.4f/%.4f Deposit %.4f Profit %.4f`
            msg = append(msg, fmt.Sprintf(tpl, short.Amount, short.AvailableAmount, short.Deposit, sprofit))
            rows = append(rows, fmt.Sprintf("L[%.0f %.4f %.1f%%]", short.Amount, sprofit, srop * 100))
            hasPosition = true
        }

        loss := false
        if lrop < -0.15 {
            ok.Trade(mctx.CloseLong, long.AvailableAmount, 0)
            loss = true
        }
        if srop < -0.15 {
            ok.Trade(mctx.CloseShort, short.AvailableAmount, 0)
            loss = true
        }

        subject := strings.Join(rows, " ")
        utils.DebugLog.Write(subject)
        if (hasPosition && n > 60) || n > 360 || loss {
            utils.SendSysMail(strings.Join(msg, "\n===============\n"), subject)
            n = 0
        }
        n++
    }
}
