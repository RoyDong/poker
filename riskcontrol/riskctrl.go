package riskcontrol

import (
    "time"
    "fmt"
    "strings"
    "dw/poker/utils"
    "dw/poker/context"
    "dw/poker/market"
    "dw/poker/market/okex"
    "math"
    "dw/poker/proto/exsync"
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
    ok := market.GetExchange(market.OkexQuarter)
    if ok == nil {
        utils.FatalLog.Write("okex exchange not found")
        return
    }
    utils.SendSysMail("start risk control")
    n := 0
    stop := false
    this.inLoop = true
    var sMaxRop = math.Inf(-1)
    var lMaxRop = math.Inf(-1)
    for this.inLoop {
        time.Sleep(2 * time.Second)
        ticker, err := ok.Tick()
        if err != nil {
            utils.WarningLog.Write(err.Error())
            continue
        }

        long := ticker.Long
        if long == nil {
            long = &exsync.Position{}
        }
        short := ticker.Short
        if short == nil {
            short = &exsync.Position{}
        }
        price := ticker.Price
        index := ticker.Index

        lprofit := market.GetProfit(long, price)
        lrop := market.GetROP(long, price)
        if lrop > lMaxRop {
            lMaxRop = lrop
        }

        sprofit := market.GetProfit(short, price)
        srop := market.GetROP(short, price)
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

        if long.AvailableAmount > 0 && lrop < -0.5 {
            ok.Trade(exsync.TradeAction_CloseLong, long.AvailableAmount, 0, 10)
            msg = append(msg, fmt.Sprintf("空单平仓 %v %.0f", exsync.TradeAction_CloseLong, long.AvailableAmount))
            stop = true
        }
        if short.AvailableAmount > 0 && srop < -0.5 {
            ok.Trade(exsync.TradeAction_CloseShort, short.AvailableAmount, 0, 10)
            msg = append(msg, fmt.Sprintf("多单平仓 %v %.0f", exsync.TradeAction_CloseShort, short.AvailableAmount))
            stop = true
        }

        subject := strings.Join(rows, " ")
        utils.DebugLog.Write(subject)
        if (hasPosition && n > 300) || n > 1000 || stop {
            utils.SendSysMail(strings.Join(msg, "\n\n"), subject)
            utils.DebugLog.Write("send mail")
            n = 0
            stop = false
        }
        n++
    }
}

