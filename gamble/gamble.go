package gamble

import (
    "dw/poker/context"
    "dw/poker/market/common"
    "dw/poker/utils"
    "dw/poker/market"
    "log"
)

type Gamble struct {
    knum int
    klines []*common.Kline
}

type indicator struct {
    //buy - sell
    dealDelta float64
    slope float64
    price float64
}

func (g *Gamble) Init(conf *context.Config) error {
    all := g.loadKlinesFromdb(10000)

    log.Println(len(all))

    n := 0
    nl := 0
    ns := 0

    ml := 0
    ms := 0
    for i := 0; i < len(all) - 11; i++ {
        //time.Sleep(time.Second)
        klines := all[i:i+6]
        testk := all[i+6]
        testk2 := all[i+7]



        ins := g.getIndicator(klines)

        if len(ins) != 5 {
            log.Println("not 5")
            continue
        }

        slopes := make([]float64, 0, 5)
        for _, in := range ins {
            slopes = append(slopes, in.slope)
        }
        prices := make([]float64, 0, 5)
        for _, in := range ins {
            prices = append(prices, in.price)
        }
        prices = append(prices, testk.AvgPrice)
        prices = append(prices, testk2.AvgPrice)


        if g.guessLong(ins) {

            nl++

            utils.DebugLog.Write("gamble guess long %v nextAvg: %f", slopes,  prices)

            if prices[5] > prices[4] || prices[6] > prices[4] {
                ml ++
            }

        } else if g.guessShort(ins) {

            ns ++
            utils.DebugLog.Write("gamble guess short %v nextAvg: %f", slopes, prices)
            if prices[5] < prices[4] || prices[6] < prices [4] {
                ms ++
            }
        } else {

            n++
        }
    }

    log.Println(n, nl, ml, ns, ms)


    return nil
}

func (g *Gamble) Run(ctx *context.Context) error {

    //ex := market.GetExchange(market.OkexQuarter)
    //klines := ex.GetKlines(10)

    return nil
}


func (g *Gamble) loadKlinesFromdb(n int) []*common.Kline {
    stmt := "select * from kline where exname = ? order by open_time asc limit ?"
    r, err := utils.MainDB.Query(stmt, market.OkexQuarter, n)
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
    return klines
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
    //前7次斜率必须小于等于0
    for i := 0; i < 2;  i++ {
        if ins[i].slope > 0 {
            return false
        }
    }
    //最近3次斜率必须大于等于0，且递增
    //主动交易 buy > sell
    for i := 2; i < 5; i++ {
        if ins[i].slope < 0 {
            return false
        }
        if ins[i].slope < ins[i-1].slope {
            return false
        }
        if ins[i].dealDelta < 0 {
            return false
        }
    }
    return true
}

func (g *Gamble) guessShort(ins []*indicator) bool {
    //前7次斜率必须大于等0
    for i := 0; i < 2;  i++ {
        if ins[i].slope < 0 {
            return false
        }
    }
    //最近3次斜率必须小于等于0，且递减
    //主动交易 buy < sell
    for i := 2; i < 5; i++ {
        if ins[i].slope > 0 {
            return false
        }
        if ins[i].slope > ins[i-1].slope {
            return false
        }
        if ins[i].dealDelta > 0 {
            return false
        }
    }
    return true
}





