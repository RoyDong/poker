package gamble

import (
    "dw/poker/context"
    "dw/poker/market/common"
    "dw/poker/utils"
    "dw/poker/market"
    "log"
    "dw/poker/protobuf/exsync"
    "time"
)

type Gamble struct {

    inloop bool
}

type indicator struct {
    //buy - sell
    dealDelta float64
    slope float64
    price float64
}

func (g *Gamble) Init(conf *context.Config) error {
    go g.play(market.OkexQuarter)
    return nil
}

func (g *Gamble) Run(ctx *context.Context) error {
    g.test(market.OkexQuarter, 2)
    g.test(market.OkexWeek, 2)

    return nil
}

func (g *Gamble) play(exname string) {
    if g.inloop {
        return
    }
    ex := market.GetExchange(exname)
    g.inloop = true
    guess := exsync.PositionType_PositionNone
    n := 1
    for g.inloop {
        time.Sleep(5 * time.Second)
        amount := float64(10)
        klines := g.loadKlinesFromdb(exname, n + 1)
        price := ex.LastnAvgPrice(50)

        //暂时没有新的分钟线

        guess = g.guess(klines, price)
        utils.DebugLog.Write("new guess %s", guess)

        long, short, err := ex.GetPosition()
        if err != nil {
            utils.FatalLog.Write("%s gamble get position error %s", exname, err.Error())
            continue
        }

        switch guess {
        case exsync.PositionType_Long:
            if short.Amount > 0 {
                ex.Trade(exsync.TradeAction_CloseShort, short.AvailableAmount, 0, 10)
            }
            if long.Amount < amount {
                ex.Trade(exsync.TradeAction_OpenLong, amount - long.Amount, 0, 10)
            }

        case exsync.PositionType_Short:
            if long.Amount > 0 {
                ex.Trade(exsync.TradeAction_CloseLong, long.AvailableAmount, 0, 10)
            }
            if short.Amount < amount {
                ex.Trade(exsync.TradeAction_OpenShort, amount - short.Amount, 0, 10)
            }

        case exsync.PositionType_PositionNone:
            if long.Amount > 0 {
                ex.Trade(exsync.TradeAction_CloseLong, long.AvailableAmount, 0, 10)
            }
            if short.Amount > 0 {
                ex.Trade(exsync.TradeAction_CloseShort, short.AvailableAmount, 0, 10)
            }
        }
        prices := make([]float64, 0, n + 1)
        for _, k := range klines {
            prices = append(prices, k.AvgPrice)
            log.Println(k.Id)
        }
        prices = append(prices, price)
        utils.DebugLog.Write("guess %s %v", guess, prices)
    }
}

func (g *Gamble) guess(klines []*common.Kline, price float64) exsync.PositionType {
    klines = append(klines, &common.Kline{AvgPrice:price})
    ins := g.getIndicator(klines)
    if g.guessLong(ins) {
        return exsync.PositionType_Long
    }
    if g.guessShort(ins) {
        return exsync.PositionType_Short
    }
    return exsync.PositionType_PositionNone
}

func (g *Gamble) test(ex string, n int) {
    all := g.loadKlinesFromdb(ex, 50000)

    var sum,nl, ns, ml, ms float64
    var lwin, swin float64
    for i := 1; i < len(all) - n - 5; i++ {
        //time.Sleep(time.Second)
        klines := all[i - 1:i + n]
        testk := all[i + n]
        testk2 := all[i+ n + 1]

        ins := g.getIndicator(klines)

        if len(ins) != n {
            log.Println("not ", i, n, len(ins))
            return
        }

        slopes := make([]float64, 0, n)
        for _, in := range ins {
            slopes = append(slopes, in.slope)
        }
        prices := make([]float64, 0, n)
        for _, in := range ins {
            prices = append(prices, in.price)
        }
        prices = append(prices, testk.AvgPrice)
        prices = append(prices, testk2.AvgPrice)

        if g.guessLong(ins) {
            nl++
            utils.DebugLog.Write("gamble guess long %v nextAvg: %f", slopes,  prices)

            w := prices[n] - prices[n-1]
            if w > 0 {//|| prices[n+1] > prices[n-1]{
                ml ++
            }
            lwin += w

        } else if g.guessShort(ins) {
            ns ++
            utils.DebugLog.Write("gamble guess short %v nextAvg: %f", slopes, prices)
            w := prices[n] - prices[n-1]
            if w < 0 {//|| prices[n+1] > prices[n-1]{
                ms ++
            }
            swin += w
        }else {
            sum ++
        }
    }

    log.Println(n, len(all),sum, nl, ml, ml/nl, ns, ms, ms/ns)
    log.Println(lwin * 6000, swin * 6000)

}

func (g *Gamble) loadKlinesFromdb(ex string, n int) []*common.Kline {
    stmt := "select * from kline where exname = ? order by open_time desc limit ?"
    r, err := utils.MainDB.Query(stmt, ex, n)
    if err != nil {
        utils.WarningLog.Write("load kline from db error %s", err.Error())
        return nil
    }
    var klines []*common.Kline
    for r.Next() {
        var k *common.Kline
        err = utils.Scan(r, &k)
        if err != nil {
            utils.WarningLog.Write("scan kline from db error %s", err.Error())
            continue
        }
        klines = append(klines, k)
    }

    l := len(klines)
    ret := make([]*common.Kline, l)
    for i, k := range klines {
        ret[l - i - 1] = k
    }
    return ret
}


/*
斜率保持连续的增长或降低，并且在最近3分钟内都保持和趋势一致的方向， 代表了价格走势的动量
同时斜率保持在一个绝对值较低的位置

 */
func (g *Gamble) getIndicator(klines []*common.Kline) []*indicator {
    indicators := make([]*indicator, 0, len(klines))
    for i := 1; i < len(klines); i++ {
        last := klines[i - 1]
        v := klines[i]
        slope := (v.AvgPrice - last.AvgPrice) / last.AvgPrice
        dealDelta := v.BuyAmount - v.SellAmount
        in := &indicator{
            price:v.AvgPrice,
            slope: slope,
            dealDelta:dealDelta,
        }
        indicators = append(indicators, in)
    }
    return indicators
}

func (g *Gamble) guessLong(ins []*indicator) bool {
    for i := 1; i < len(ins); i++ {
        if ins[i].slope < ins[i-1].slope {
            return false
        }
    }
    return ins[len(ins) - 1].slope > 0
}

func (g *Gamble) guessShort(ins []*indicator) bool {
    for i := 1; i < len(ins); i++ {
        if ins[i].slope > ins[i-1].slope {
            return false
        }
    }
    return ins[len(ins) - 1].slope < 0
}





