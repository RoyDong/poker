package arbitrage

import (
    "time"
    "fmt"
    "github.com/roydong/gmvc"
    "log"
    "sync"
    "math"
)

const (
    StateStop         = 0
    StateOpen         = 1
    StateClose        = 2
)

type Hedge struct {
    zuo, you *Exchange
    long, short *Exchange

    tradeAmount float64
    maxTradeAmount float64
    minTradeMargin float64

    state         int
    started       time.Time
    counter       int

    margin *averager

    test          bool
}


func NewHedge(zuo, you *Exchange) *Hedge {
    hg := &Hedge{
        zuo: zuo,
        you: you,

        margin: newAverager(300),

        minTradeMargin: 5,
    }

    conf := gmvc.Store.Tree("config.hedge")
    if v, has := conf.Float("min_trade_margin"); has {
        hg.minTradeMargin = v
    }
    if v, has := conf.Float("max_trade_amount"); has {
        hg.maxTradeAmount = v
    }

    if v, has := conf.Int("test"); has {
        if v == 1 {
            hg.test = true
        } else {
            hg.test = false
        }
    }

    return hg
}

func (hg *Hedge) Start() {
    hg.zuo.SyncBalance()
    hg.you.SyncBalance()
    amount := hg.zuo.amount + hg.you.amount
    money := hg.zuo.money + hg.you.money
    gmvc.Logger.Println(fmt.Sprintf("hedge start %v <---> %v", hg.zuo.Name(), hg.you.Name()))
    gmvc.Logger.Println(fmt.Sprintf("%v: %.4f btc, %.2f cny", hg.zuo.Name(), hg.zuo.amount, hg.zuo.money))
    gmvc.Logger.Println(fmt.Sprintf("%v: %.4f btc, %.2f cny", hg.you.Name(), hg.you.amount, hg.you.money))
    gmvc.Logger.Println(fmt.Sprintf("total: %.4f btc, %.2f cny", amount, money))
    gmvc.Logger.Println(fmt.Sprintf("max trade: %.4f btc, min margin: %.2f", hg.maxTradeAmount, hg.minTradeMargin))
    gmvc.Logger.Println("--------")

    hg.state = StateClose
    hg.started = time.Now()
    hg.counter = 0

    go hg.calcMargins()
    go hg.arbitrage(500 * time.Millisecond)
}

func (hg *Hedge) Stop() {
    hg.state = StateStop
}

func (hg *Hedge) calcMargins() {
    wg := &sync.WaitGroup{}
    for now := range time.Tick(1 * time.Second) {
        if hg.state == StateStop {
            break
        }
        idx := int64(now.Second())
        wg.Add(2)
        go func() {
            hg.zuo.calcMa()
            wg.Done()
        }()
        go func() {
            hg.you.calcMa()
            wg.Done()
        }()
        wg.Wait()

        hg.margin.Add(idx, hg.you.ma - hg.zuo.ma)
        log.Println(fmt.Sprintf("%.2f <= %.2f(%v) => %.2f", hg.margin.Min(),
            hg.margin.Avg(), hg.margin.Len(), hg.margin.Max()))
    }
}

func (hg *Hedge) arbitrage(interval time.Duration) {
    wg := &sync.WaitGroup{}
    for hg.state > StateStop {
        time.Sleep(interval)
        if hg.margin.Len() < 50 {
            continue
        }

        wg.Add(2)
        go func() {
            hg.zuo.SyncDepth()
            wg.Done()
        }()
        go func() {
            hg.you.SyncDepth()
            wg.Done()
        }()
        wg.Wait()

        if len(hg.zuo.lastAsks) == 0 {
            gmvc.Logger.Println(hg.zuo.name + " depth is empty")
            continue
        }
        if len(hg.you.lastAsks) == 0 {
            gmvc.Logger.Println(hg.you.name + " depth is empty")
            continue
        }

        zuoBuyPrice, zuoBuyAmount := hg.zuo.GetAskDepth(1)
        zuoSellPrice, zuoSellAmount := hg.zuo.GetBidDepth(1)

        youBuyPrice, youBuyAmount := hg.you.GetAskDepth(1)
        youSellPrice, youSellAmount := hg.you.GetBidDepth(1)

        var margin float64
        if hg.state == StateClose {

            //尝试判断是否可以右手做空(左手多), 以右手的最近买单价 - 左手的卖单价(margin)和(min max avg)相关参数比较
            margin = youSellPrice - zuoBuyPrice
            //log.Println(fmt.Sprintf("margin: sell %.2f max %.2f", margin, hg.margin.Max()))

            //满足最小差价限制,并且超过最大差价
            if margin >= hg.margin.Avg() + hg.minTradeMargin && margin >= hg.margin.Max() {
                gmvc.Logger.Println(fmt.Sprintf("open positoin(youSell - zuoBuy %.2f):", margin ))
                hg.tradeAmount = math.Min(math.Min(youSellAmount, zuoBuyAmount), hg.maxTradeAmount)
                hg.openPosition(hg.you, youSellPrice, hg.zuo, zuoBuyPrice)
                continue
            }

            //尝试判断是否可以左手做空(右手多), 以右手的最近卖单价 - 左手的买单价(margin)和(min max avg)相关参数比较
            margin = youBuyPrice - zuoSellPrice
            //log.Println(fmt.Sprintf("margin: buy %.2f min %.2f", margin, hg.margin.Min()))

            //满足最小差价限制,并且低于最小差价
            if margin <= hg.margin.Avg() - hg.minTradeMargin && margin <= hg.margin.Min() {
                gmvc.Logger.Println(fmt.Sprintf("open position(youBuy - zuoSell %.2f):", margin))
                hg.tradeAmount = math.Min(math.Min(youBuyAmount, zuoSellAmount), hg.maxTradeAmount)
                hg.openPosition(hg.zuo, zuoSellPrice, hg.you, youBuyPrice)
                continue
            }

        } else {

            //如果是右手做空
            if (hg.short.name == hg.you.name) {
                margin = youBuyPrice - zuoSellPrice

                //差价低于平均差价即可平仓
                if margin <= hg.margin.Avg() {
                    gmvc.Logger.Println(fmt.Sprintf("close position(youBuy - zuoSell %.2f):", margin))
                    hg.closePosition(youBuyPrice, zuoSellPrice)
                }

            //如果是左手做空的
            } else {
                margin = youSellPrice - zuoBuyPrice

                //差价高于平均差价即可平仓
                if margin >= hg.margin.Avg() {
                    gmvc.Logger.Println(fmt.Sprintf("close position(youSell - zuoBuy %.2f):", margin))
                    hg.closePosition(zuoBuyPrice, youSellPrice)
                }
            }
        }
    }
}

func (hg *Hedge) openPosition(short *Exchange, shortSellPrice float64, long *Exchange, longBuyPrice float64) {
    if hg.test {
        return
    }

    var sorder, lorder Order
    /*
    if short.Name() == "huobi" {
        sorder = hg.openShort(short, shortSellPrice)
        if sorder.DealAmount > 0.0001 {
            lorder = hg.openLong(long, longBuyPrice)
        } else {
            return
        }
    } else {
        lorder = hg.openLong(long, longBuyPrice)
        if lorder.DealAmount > 0.0001 {
            sorder = hg.openShort(short, shortSellPrice)
        } else {
            return
        }
    }
    */

    wg := &sync.WaitGroup{}
    wg.Add(2)
    go func() {
        hg.long = long
        lorder = long.Trade(OpenLongPosition, hg.tradeAmount, longBuyPrice)
        wg.Done()
    }()
    go func() {
        hg.short = short
        sorder = short.Trade(OpenShortPosition, hg.tradeAmount, shortSellPrice)
        wg.Done()
    }()
    wg.Wait()

    /*
    检查下单结果，补齐对冲敞口
    1. 当空头交易过多，则空方市价平仓买入差额
    2. 当多头交易过多，则多方市价平仓卖出差额
     */
    if sorder.DealAmount > lorder.DealAmount {
        delta := sorder.DealAmount - lorder.DealAmount
        if order := short.TradeAll(CloseShortPosition, delta); order.DealAmount > 0 {
            sorder.DealAmount -= order.DealAmount
        }
    } else if lorder.DealAmount > sorder.DealAmount {
        delta := lorder.DealAmount - sorder.DealAmount
        if order := long.TradeAll(CloseLongPosition, delta); order.DealAmount > 0 {
            lorder.DealAmount -= order.DealAmount
        }
    }

    short.amountChange -= sorder.DealAmount
    long.amountChange += lorder.DealAmount

    //交易统计
    gmvc.Logger.Println(fmt.Sprintf("   short: %v - %.4f btc + %.2f(%.2f) cny",
        short.name, sorder.DealAmount, shortSellPrice, sorder.AvgPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v + %.4f btc - %.2f(%.2f) cny",
        long.name, lorder.DealAmount, longBuyPrice, lorder.AvgPrice))

    if sorder.DealAmount <= 0.0001 || lorder.DealAmount <= 0.0001 {
        gmvc.Logger.Println("closed")
    } else {
        hg.state = StateOpen
        gmvc.Logger.Println("openned")
    }
}

func (hg *Hedge) closePosition(buyPrice, sellPrice float64) {
    if hg.test {
        return
    }

    wg := &sync.WaitGroup{}
    wg.Add(2)

    /*
    立即使用对价完成所有交易
     */
    var sorder, lorder Order
    go func() {
        sorder = hg.short.TradeAll(CloseShortPosition, -hg.short.amountChange)
        hg.short.amountChange += sorder.DealAmount
        wg.Done()
    }()
    go func() {
        lorder = hg.long.TradeAll(CloseLongPosition, hg.long.amountChange)
        hg.short.amountChange -= lorder.DealAmount
        wg.Done()
    }()
    wg.Wait()

    hg.state = StateClose
    hg.counter++

    //交易统计
    gmvc.Logger.Println(fmt.Sprintf("   short: %v + %.4f btc - %.2f(%.2f) cny",
        hg.short.name, sorder.DealAmount, buyPrice, sorder.AvgPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v - %.4f btc + %.2f(%.2f) cny",
        hg.long.name, lorder.DealAmount, sellPrice, lorder.AvgPrice))
    gmvc.Logger.Println("")

    now := time.Now()
    d := time.Unix(now.Unix() - hg.started.Unix() - 28800, 0)
    gmvc.Logger.Println(fmt.Sprintf("result: %.4f btc, %.2f cny, %v/%v",
                                    hg.long.amountChange + hg.short.amountChange,
                                    hg.long.moneyChange+ hg.short.moneyChange,
                                    hg.counter, d.Format("15:04:05")))
    hg.zuo.SyncBalance()
    hg.you.SyncBalance()
    gmvc.Logger.Println(fmt.Sprintf("    %v: %.4f btc, %.2f cny", hg.zuo.Name(), hg.zuo.amount, hg.zuo.money))
    gmvc.Logger.Println(fmt.Sprintf("    %v: %.4f btc, %.2f cny", hg.you.Name(), hg.you.amount, hg.you.money))
    gmvc.Logger.Println("")
}
