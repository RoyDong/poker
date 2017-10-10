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
    "math"
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
    stop := false
    this.inLoop = true
    var sMaxRop = math.Inf(-1)
    var lMaxRop = math.Inf(-1)
    for this.inLoop {
        <- time.After(10 * time.Second)
        wg := sync.WaitGroup{}
        wg.Add(2)
        var long, short mctx.Position
        var index float64
        var err error
        go func() {
            long, short, err = ok.GetPosition()
            wg.Done()
        }()
        go func() {
            index, _ = ok.GetIndex()
            wg.Done()
        }()
        wg.Wait()

        if err != nil {
            utils.WarningLog.Write("req api error %s", err.Error())
            continue
        }

        price := ok.LastnAvgPrice(5)
        lprofit := long.GetProfit(price)
        lrop := long.GetROP(price)
        if lrop > lMaxRop {
            lMaxRop = lrop
        }

        sprofit := short.GetProfit(price)
        srop := short.GetROP(price)
        if srop > sMaxRop {
            sMaxRop = srop
        }

        usdprice := okex.FutureBTC_USD(price)
        usdindex := okex.FutureBTC_USD(index)

        rows := make([]string, 0, 3)
        rows = append(rows, fmt.Sprintf("P[%.2f %.1f%%]", usdprice, (usdprice - usdindex) / usdindex * 100))

        msg := make([]string, 0, 2)
        hasPosition := false
        if long.Amount > 0 {
            tpl := `空头(usd long) %.0f/%.0f Deposit %.4f Profit %.4f`
            msg = append(msg, fmt.Sprintf(tpl, long.Amount, long.AvailableAmount, long.Deposit, lprofit))
            rows = append(rows, fmt.Sprintf("S[%.0f %.4f %.1f%%]", long.Amount, lprofit, lrop * 100))
            hasPosition = true
        } else {
            lMaxRop = 0
        }
        if short.Amount > 0 {
            tpl := `多头(usd short) %.0f/%.0f Deposit %.4f Profit %.4f`
            msg = append(msg, fmt.Sprintf(tpl, short.Amount, short.AvailableAmount, short.Deposit, sprofit))
            rows = append(rows, fmt.Sprintf("L[%.0f %.4f %.1f%%]", short.Amount, sprofit, srop * 100))
            hasPosition = true
        } else {
            sMaxRop = 0
        }

        //回调20%止盈  亏损15%止损
        if long.AvailableAmount > 0 && ((lrop < lMaxRop - 0.3) || lrop < -0.15) {
            ok.Trade(mctx.CloseLong, long.AvailableAmount, 0)
            msg = append(msg, fmt.Sprintf("空单平仓 %v %.0f", mctx.CloseLong, long.AvailableAmount))
            stop = true
        }
        if short.AvailableAmount > 0 && ((srop < sMaxRop - 0.3) || srop < -0.15) {
            ok.Trade(mctx.CloseShort, short.AvailableAmount, 0)
            msg = append(msg, fmt.Sprintf("多单平仓 %v %.0f", mctx.CloseShort, short.AvailableAmount))
            stop = true
        }

        subject := strings.Join(rows, " ")
        utils.DebugLog.Write(subject)
        if (hasPosition && n > 60) || n > 300 || stop {
            utils.SendSysMail(strings.Join(msg, "\n\n"), subject)
            utils.DebugLog.Write("send mail")
            n = 0
            stop = false
        }
        n++
    }
}

