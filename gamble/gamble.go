package gamble

import (
    "dw/poker/context"
    "dw/poker/market/common"
    "dw/poker/utils"
    "dw/poker/market"
    "log"
    "dw/poker/protobuf/exsync"
    "time"
    "dw/poker/ml"
)

const (
    GuessLong = 1
    GuessLongPending = 2

    GuessShort = 3
    GuessShortPending = 4

    GuessNone = 5
)

type Gamble struct {

}

type indicator struct {
    //buy - sell
    dealDelta float64
    slope int
    price float64
}

func (g *Gamble) Init(conf *context.Config) error {
    g.test(market.OkexWeek, 2)
    g.test(market.OkexQuarter, 2)
    //go g.play(market.OkexWeek)
    //go g.play(market.OkexQuarter)
    return nil
}

func (g *Gamble) Run(ctx *context.Context) error {
    g.test(market.OkexQuarter, 3)
    g.test(market.OkexWeek, 3)

    return nil
}

func (g *Gamble) train(exname string) {
    ex := market.GetExchange(exname)
    sample := make([]*ml.LabeledPoint, 0)

    klines := g.loadKlinesFromdb(exname, 50000)
    slopes := make([]int, 0, len(klines))

    for i := 1; i < len(klines); i++ {
        slope
    }



}


func (g *Gamble) play(exname string) {
    ex := market.GetExchange(exname)
    guess := exsync.PositionType_PositionNone
    for {
        time.Sleep(5 * time.Second)
        amount := float64(10)
        klines := g.loadKlinesFromdb(exname, 3)
        price := ex.LastnAvgPrice(10)


        guess = g.guess(append(klines, &common.Kline{AvgPrice:price}))
        utils.DebugLog.Write("guess %s", guess)

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
    }
}

func (g *Gamble) guess(klines []*common.Kline) exsync.PositionType {
    ins := g.getIndicator(klines)
    var v []int
    for _, in := range ins {
        v = append(v, in.slope)
    }
    utils.DebugLog.Write("slopes %v", v)
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
    var maxs float64
    var mins float64
    var sum,nl, ns, ml, ms float64
    var lwin, swin float64
    for i := 1; i < len(all) - n - 5; i++ {
        //time.Sleep(time.Second)
        klines := all[i - 1:i + n]
        testk := all[i + n]
        testk2 := all[i+ n + 1]

        prices := make([]float64, 0)
        for _, k := range klines {
            prices = append(prices, k.AvgPrice)
        }
        ins := g.getIndicator(klines)

        if len(ins) != n {
            log.Println("not ", i, n, len(ins))
            return
        }

        slopes := make([]int, 0, n)
        for _, in := range ins {
            slopes = append(slopes, in.slope)
        }
        prices = append(prices, testk.AvgPrice)
        prices = append(prices, testk2.AvgPrice)

        if g.guessLong(ins) {
            nl++
            //utils.DebugLog.Write("gamble guess long %v nextAvg: %f", slopes,  prices)

            w := prices[n+2] - prices[n + 1]
            if w > 0 {
                ml ++
            }
            lwin += w

        } else if g.guessShort(ins) {
            ns ++
            //utils.DebugLog.Write("gamble guess short %v nextAvg: %f", slopes, prices)
            w := prices[n+2] - prices[n+1]
            if w < 0 {
                ms ++
            }
            swin += w
        }else {
            sum ++
        }
    }

    log.Println("max min", maxs, mins)
    log.Println("all", n, len(all), sum, 1 - sum / float64(len(all)))
    log.Println(nl, ml, ml/nl, ns, ms, ms/ns)
    log.Println(lwin * 6000 / nl, lwin * 6000, swin * 6000 / ns, swin * 6000)

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
    for i := 0; i < len(klines) / 2; i++ {
        j := len(klines) - i - 1
        klines[i], klines[j] = klines[j], klines[i]
    }
    return klines
}

var maxSlope = 0.05

/*
斜率保持连续的增长或降低，并且在最近3分钟内都保持和趋势一致的方向， 代表了价格走势的动量
同时斜率保持在一个绝对值较低的位置
 */
func (g *Gamble) getIndicator(klines []*common.Kline) []*indicator {
    indicators := make([]*indicator, 0, len(klines))
    for i := 1; i < len(klines); i++ {
        v1 := klines[i-1]
        v2 := klines[i]
        s := (v2.AvgPrice - v1.AvgPrice) / v1.AvgPrice
        slope := int(s / maxSlope * 100)
        in := &indicator{
            price:v2.AvgPrice,
            slope: slope,
            dealDelta:v2.BuyAmount - v2.SellAmount,
        }
        indicators = append(indicators, in)
    }
    return indicators
}

func (g *Gamble) guessLong(ins []*indicator) bool {
    for i := 1; i < len(ins); i++ {
        if ins[i].slope <= ins[i-1].slope {
            return false
        }
    }
    slope := ins[len(ins) - 1].slope
    return slope > 5 && slope < 14
}

func (g *Gamble) guessShort(ins []*indicator) bool {
    for i := 1; i < len(ins); i++ {
        if ins[i].slope >= ins[i-1].slope {
            return false
        }
    }
    slope := ins[len(ins) - 1].slope
    return slope < -5 && slope > -14
}



