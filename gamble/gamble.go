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

type Gamble struct { }

type indicator struct {
    //buy - sell
    dealDelta float64
    slope int
    highPrice float64
    lowPrice float64
    price float64
    amount float64
    gradient float64
}

func (g *Gamble) Init(conf *context.Config) error {
    ok := market.GetExchange(market.OkexQuarter)
    sm := &SpreadMargin{}
    sm.tradeAmount = 10
    sm.margin = 0.0009
    sm.Run(ok)


    //g.train(market.OkexQuarter)
    n := 10
    //g.test(market.OkexWeek, n)
    g.test(market.OkexQuarter, n)
    //go g.play(market.OkexWeek, n)
    //go g.play(market.OkexQuarter, n)
    return nil
}

func (g *Gamble) Run(ctx *context.Context) error {
    g.test(market.OkexQuarter, 3)
    g.test(market.OkexWeek, 3)

    return nil
}

func (g *Gamble) train(exname string) {
    //ex := market.GetExchange(exname)
    klines := g.loadKlinesFromdb(exname, 100000)
    log.Println(len(klines))
    slopes := make([]int, 0, len(klines))
    //slopes = append(slopes, 0)
    for i := 1; i < len(klines) - 100; i++ {
        k1 := klines[i - 1]
        k2 := klines[i]
        slopes = append(slopes, g.getSlope(k1.AvgPrice, k2.AvgPrice))
    }

    sample := make([]*ml.LabeledPoint, 0, len(klines))
    fnum := 100
    for i := 0; i < len(slopes) - fnum - 2; i++ {
        label := 0.0
        if slopes[i+fnum] < 0 {
            label = 1
        }
        if slopes[i+fnum] > 0 {
            //label = 1
        }
        dense := make([]float64, fnum * 40)
        for fi, v := range slopes[i:i+fnum] {
            if v >= 20 {
                v = 19
            } else if v < -20 {
                v = -20
            }
            v = v + 20
            dense[fi * 20 + v] = 1
        }
        lp := &ml.LabeledPoint{
            Label:label,
            Features:ml.NewVectorWithDense(dense),
        }
        sample = append(sample, lp)
    }

    lr := &ml.LogisticRegression{}
    lr.BatchGradAscent(sample[:26000], 400)

    rtp := make([]float64, 11)
    rtn := make([]float64, 11)
    testSample := sample[26000:]
    for _, lp := range testSample {
        p := lr.Predict(lp.Features)

        i := int(p * 10)
        if lp.Label > 0 {
            rtp[i] = rtp[i] + 1
        } else {
            rtn[i] = rtn[i] + 1
        }
    }

    log.Println(len(testSample), 26000 / len(testSample))
    for i := 10; i >= 0; i-- {
        log.Println(i, rtp[i], rtn[i], rtp[i] / (rtp[i] + rtn[i]))
    }

}

func (g *Gamble) lrTest() {

}


func (g *Gamble) play(exname string, num int) {
    ex := market.GetExchange(exname)
    guess := exsync.PositionType_PositionNone
    for {
        time.Sleep(5 * time.Second)
        amount := float64(10)
        //klines := g.loadKlinesFromdb(exname, num)
        price := ex.LastnAvgPrice(10)

        guess = exsync.PositionType_PositionNone //g.guess(append(klines, &common.Kline{AvgPrice:price}))
        utils.DebugLog.Write("guess %s", guess)

        long, short, err := ex.GetPosition()
        if err != nil {
            utils.FatalLog.Write("%s gamble get position error %s", exname, err.Error())
            continue
        }

        switch guess {
        case exsync.PositionType_Long:
            if short.Amount > 0 {
                ex.Trade(exsync.TradeAction_CloseShort, short.AvailableAmount, 0)
            }
            if long.Amount < amount {
                ex.Trade(exsync.TradeAction_OpenLong, amount - long.Amount, price)
            }

        case exsync.PositionType_Short:
            if long.Amount > 0 {
                ex.Trade(exsync.TradeAction_CloseLong, long.AvailableAmount, 0)
            }
            if short.Amount < amount {
                ex.Trade(exsync.TradeAction_OpenShort, amount - short.Amount, price)
            }

        case exsync.PositionType_PositionNone:
            if long.Amount > 0 {
                ex.Trade(exsync.TradeAction_CloseLong, long.AvailableAmount, 0)
            }
            if short.Amount > 0 {
                ex.Trade(exsync.TradeAction_CloseShort, short.AvailableAmount, 0)
            }
        }
        ex.CancelAllOrders()
    }
}

func (g *Gamble) guess(candles []*common.Candle) exsync.PositionType {
    ins := g.getIndicator(candles)
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

var fee = 0.0003

func (g *Gamble) test(ex string, n int) {
    //all := g.loadKlinesFromdb(ex, 100000)
    ok := market.GetExchange(market.OkexQuarter)
    all := ok.LoadCandles(5, 60)
    var sum,nl, ns, ml, ms float64
    var lwin, swin float64
    for i := 1; i < len(all) - n - 20; {
        //time.Sleep(time.Second)
        candles := all[i - 1:i + n]

        prices := make([]float64, 0)
        for _, k := range candles {
            prices = append(prices, k.AvgPrice)
        }
        ins := g.getIndicator(candles)

        if len(ins) != n {
            log.Println("not ", i, n, len(ins))
            return
        }

        slopes := make([]int, 0, n)
        for _, in := range ins {
            slopes = append(slopes, in.slope)
        }
        prices = append(prices, all[i+n].AvgPrice)
        prices = append(prices, all[i+n+1].AvgPrice)
        prices = append(prices, all[i+n+2].AvgPrice)
        prices = append(prices, all[i+n+3].AvgPrice)
        prices = append(prices, all[i+n+4].AvgPrice)
        prices = append(prices, all[i+n+5].AvgPrice)
        prices = append(prices, all[i+n+6].AvgPrice)
        prices = append(prices, all[i+n+7].AvgPrice)
        prices = append(prices, all[i+n+8].AvgPrice)
        prices = append(prices, all[i+n+9].AvgPrice)

        if g.guessShort(ins) {
            ns ++
            //utils.DebugLog.Write("gamble guess short %v nextAvg: %f", slopes, prices)
            var round float64
            for m := 0; m < 10; m ++ {
                w := prices[n + m] - prices[n + m - 1]
                round += w
                i++
                if w > 0 {
                    break
                }
            }
            if round + fee < 0 {
                ms++
            }
            swin = swin + round + fee


        } else if g.guessLong(ins) {
            nl++
            //utils.DebugLog.Write("gamble guess long %v nextAvg: %f", slopes,  prices)
            var round float64
            for m := 0; m < 10; m ++ {
                w := prices[n + m] - prices[n + m - 1]
                round += w
                i++
                if w < 0 {
                    break
                }
            }

            if round - fee > 0 {
                ml++
            }

            lwin = lwin + round - fee
        } else {
            i++
            sum ++
        }
    }

    log.Println("grad conf", minGrad, burstGrad, len(all), n)
    log.Println("recall", sum, 1 - sum / float64(len(all)), len(all) - int(sum))
    log.Println("long", nl, ml, ml/nl)
    log.Println(lwin  / nl, lwin )
    log.Println("short", ns, ms, ms/ns)
    log.Println(swin / ns, swin )
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
func (g *Gamble) getIndicator(candles []*common.Candle) []*indicator {
    indicators := make([]*indicator, 0, len(candles))
    for i := 1; i < len(candles); i++ {
        v1 := candles[i-1]
        v2 := candles[i]
        slope := g.getSlope(v1.AvgPrice, v2.AvgPrice)
        grad := (v2.AvgPrice - v1.AvgPrice) / v1.AvgPrice
        in := &indicator{
            highPrice:v2.HighPrice,
            lowPrice:v2.LowPrice,
            price:v2.AvgPrice,
            amount:v2.Amount,
            slope: slope,
            dealDelta:v2.BuyAmount - v2.SellAmount,
            gradient: grad,
        }
        indicators = append(indicators, in)
    }
    return indicators
}

func (g *Gamble) getSlope(p1, p2 float64) int {
    s := (p2 - p1) / p1
    return int(s / maxSlope * 100)
}


var minGrad = 0.003
var burstGrad = 0.005

func (g *Gamble) guessLong(ins []*indicator) bool {
    for i := 1; i < len(ins); i++ {
        if ins[i].gradient <= ins[i - 1].gradient {
            return false
        }
        if ins[i].amount <= ins[i - 1].amount {
            return false
        }
    }
    if ins[len(ins) - 1].gradient > burstGrad {
        return true
    }
    return false
}

func getGrad(p1, p2 float64) float64 {
    return (p1 - p2) / p2
}

func (g *Gamble) guessShort(ins []*indicator) bool {
    for i := 1; i < len(ins); i++ {
        if ins[i].gradient >= ins[i - 1].gradient {
            return false
        }
        if ins[i].amount >= ins[i - 1].amount {
            return false
        }
    }
    if ins[len(ins) - 1].gradient < -burstGrad {
        return true
    }
    return false
}



