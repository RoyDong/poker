package market

import (
    "time"
    "fmt"
    "github.com/roydong/gmvc"
    "log"
    "sync"
)

type Hedger struct {
    zuo *Market
    you *Market

    zuoBuyAmount float64
    youBuyAmount float64

    short *Market
    long *Market

    tradeAmount float64
    minTradeMargin float64
    minAvg, midAvg, maxAvg *averager

    stoped        bool
    state         int

    started       time.Time
    tradeNum      int

    tcny        float64
    cny         float64
    btc         float64
}


func NewHedger(zuo, you *Market) *Hedger {
    hg := &Hedger{
        zuo: zuo,
        you: you,

        minAvg: newAverager(20),
        midAvg: newAverager(100),
        maxAvg: newAverager(20),

        state: StateClose,
    }

    zuo.SyncBalance()
    you.SyncBalance()

    conf := gmvc.Store.Tree("config.hedger")
    hg.minTradeMargin, _ = conf.Float("min_trade_margin")
    hg.tradeAmount, _ = conf.Float("trade_amount")

    hg.zuoBuyAmount = hg.tradeAmount
    hg.youBuyAmount = hg.tradeAmount

    return hg
}

func (hg *Hedger) Start() {
    hg.stoped = false

    hg.tradeNum = 0
    hg.started = time.Now()

    btc := hg.zuo.btc + hg.you.btc
    cny := hg.zuo.cny + hg.you.cny
    gmvc.Logger.Println("--------")
    gmvc.Logger.Println(fmt.Sprintf("btc: %.2f, cny: %.2f", btc, cny))

    go hg.updateMargins(500 * time.Millisecond)
    go hg.arbitrage(500 * time.Millisecond)
}

func (hg *Hedger) Stop() {
    hg.stoped = true
}

func (hg *Hedger) updateMargins(interval time.Duration) {
    wg := &sync.WaitGroup{}
    for _ = range time.Tick(interval) {
        var zuoTicker, youTicker Ticker
        wg.Add(2)
        go func() {
            zuoTicker = hg.zuo.LastTicker()
            wg.Done()
        }()
        go func() {
            youTicker = hg.you.LastTicker()
            wg.Done()
        }()
        wg.Wait()
        if zuoTicker.Last <= 0 || youTicker.Last <= 0 {
            continue
        }

        idx := zuoTicker.Time
        margin := youTicker.Last - zuoTicker.Last

        if hg.midAvg.Len() > 0 {
            if margin <= hg.midAvg.Avg() - hg.minTradeMargin {
                hg.minAvg.Add(idx, margin)
            } else if margin >= hg.midAvg.Avg() + hg.minTradeMargin {
                hg.maxAvg.Add(idx, margin)
            }
        }

        if overflow, idx := hg.midAvg.Add(idx, margin); overflow {
            hg.minAvg.CutTail(idx)
            hg.maxAvg.CutTail(idx)
        }

        log.Println(fmt.Sprintf("%.3f <= %.3f(%.3f) => %.3f",
            hg.minAvgMargin(), hg.midAvg.Avg(), margin, hg.maxAvgMargin()))
    }
}

func (hg *Hedger) minAvgMargin() float64 {
    if hg.minAvg.Len() > 0 {
        return hg.minAvg.Avg()
    }
    return hg.midAvg.Avg() - hg.minTradeMargin
}

func (hg *Hedger) maxAvgMargin() float64 {
    if hg.maxAvg.Len() > 0 {
        return hg.maxAvg.Avg()
    }
    return hg.midAvg.Avg() + hg.minTradeMargin
}

func (hg *Hedger) arbitrage(interval time.Duration) {
    for _ = range time.Tick(interval) {
        if hg.midAvg.Len() < 20 {
            gmvc.Logger.Println("margin list is less than 10")
            continue
        }

        hg.zuo.UpdateDepth()
        hg.you.UpdateDepth()
        if len(hg.zuo.lastAsks) == 0 {
            gmvc.Logger.Println(hg.zuo.name + " depth is empty")
            continue
        }
        if len(hg.you.lastAsks) == 0 {
            gmvc.Logger.Println(hg.you.name + " depth is empty")
            continue
        }

        zuoBuyPrice := hg.zuo.GetBuyPrice(hg.tradeAmount)
        zuoSellPrice := hg.zuo.GetSellPrice(hg.tradeAmount)

        youBuyPrice := hg.you.GetBuyPrice(hg.tradeAmount)
        youSellPrice := hg.you.GetSellPrice(hg.tradeAmount)

        var margin float64
        if hg.state == StateClose {

            //尝试判断是否可以右手做空(左手多), 以右手的最近买单价 - 左手的卖单价(margin)和(min max avg)相关参数比较
            margin = youSellPrice - zuoBuyPrice
            //log.Println(fmt.Sprintf("youSell - zuoBuy %.2f(%.2f)", margin, margin - hg.avgMargin))

            //满足最小差价条件,并且超过最大差价
            if margin >= hg.maxAvgMargin() {
                gmvc.Logger.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))
                hg.openPosition(hg.you, youSellPrice, hg.zuo, zuoBuyPrice)
                continue
            }

            //尝试判断是否可以左手做空(右手多), 以右手的最近卖单价 - 左手的买单价(margin)和(min max avg)相关参数比较
            margin = youBuyPrice - zuoSellPrice
            //log.Println(fmt.Sprintf("youBuy - zuoSell %.2f(%.2f)", margin, margin - hg.avgMargin))

            //满足最小差价条件,并且低于最小差价
            if margin <= hg.minAvgMargin() {
                gmvc.Logger.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))
                hg.openPosition(hg.zuo, zuoSellPrice, hg.you, youBuyPrice)
                continue
            }

        } else {

            //如果是右手做空
            if (hg.short.name == hg.you.name) {
                margin = youBuyPrice - zuoSellPrice
                //log.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))

                //差价低于平均差价即可平仓
                if margin <= hg.midAvg.Avg() {
                    gmvc.Logger.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))
                    hg.closePosition(youBuyPrice, zuoSellPrice)
                }

            //如果是左手做空的
            } else {
                margin = youSellPrice - zuoBuyPrice
                //log.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))

                //差价高于平均差价即可平仓
                if margin >= hg.midAvg.Avg() {
                    gmvc.Logger.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))
                    hg.closePosition(zuoBuyPrice, youSellPrice)
                }
            }
        }
    }
}

func (hg *Hedger) openPosition(short *Market, shortSellPrice float64, long *Market, longBuyPrice float64) {
    var sid, lid int64
    if short.name == "huobi" {
        sid = hg.openShort(short, shortSellPrice)
        if sid == 0 {
            return
        }
        lid = hg.openLong(long, longBuyPrice)
    } else {
        lid = hg.openLong(long, longBuyPrice)
        if lid == 0 {
            return
        }
        sid = hg.openShort(short, shortSellPrice)
    }

    hg.state = StateOpen

    //交易统计
    var sorder, lorder Order
    for _ = range time.Tick(500 * time.Millisecond) {
        sorder = short.OrderInfo(sid)
        if sorder.Status == 2 {
            short.lastSell = sorder.DealAmount
            break
        }
    }

    for _ = range time.Tick(500 * time.Millisecond) {
        lorder = long.OrderInfo(lid)
        if lorder.Status == 2 {
            long.lastBuy = lorder.DealAmount
            break
        }
    }

    hg.btc += lorder.DealAmount - hg.tradeAmount
    hg.cny += sorder.AvgPrice - lorder.AvgPrice
    hg.tcny += shortSellPrice - longBuyPrice

    gmvc.Logger.Println("open position:")
    gmvc.Logger.Println(fmt.Sprintf("   short: %v - %.2f(%.2f) btc, + %.2f(%.2f) cny",
        short.name, hg.tradeAmount, sorder.DealAmount, shortSellPrice, sorder.AvgPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v + %.2f(%.2f) btc, - %.2f(%.2f) cny",
        long.name, hg.tradeAmount, lorder.DealAmount, longBuyPrice, lorder.AvgPrice))
    gmvc.Logger.Println("")
}

func (hg *Hedger) openShort(short *Market, sellPrice float64) int64 {
    amount := hg.tradeAmount
    if short.lastBuy > 0 {
        amount = short.lastBuy
    }
    id := short.Sell(amount)
    hg.short = short
    return id
}

func (hg *Hedger) openLong(long *Market, buyPrice float64) int64 {
    id := long.Buy(hg.tradeAmount * buyPrice)
    hg.long = long
    return id
}


func (hg *Hedger) closePosition(buyPrice, sellPrice float64) {
    var sid, lid int64
    if hg.short.name == "huobi" {
        sid = hg.closeShort(buyPrice)
        if sid == 0 {
            return
        }
        lid = hg.closeLong(sellPrice)
    } else {
        lid = hg.closeLong(sellPrice)
        if lid == 0 {
            return
        }
        sid = hg.closeShort(buyPrice)
    }
    hg.state = StateClose
    hg.tradeNum++

    //交易统计
    var sorder, lorder Order
    for _ = range time.Tick(500 * time.Millisecond) {
        sorder := hg.short.OrderInfo(sid)
        if sorder.Status == 2 {
            hg.short.lastBuy = sorder.DealAmount
            break
        }
    }

    for _ = range time.Tick(500 * time.Millisecond) {
        lorder := hg.long.OrderInfo(lid)
        if lorder.Status == 2 {
            hg.long.lastSell = lorder.DealAmount
            break
        }
    }

    hg.btc += sorder.DealAmount - lorder.DealAmount
    hg.cny += lorder.AvgPrice - sorder.AvgPrice
    hg.tcny += sellPrice - buyPrice

    gmvc.Logger.Println("close position:")
    gmvc.Logger.Println(fmt.Sprintf("   short: %v + %.2f(%.2f) btc, - %.2f(%.2f) cny",
        hg.short.name, hg.tradeAmount, sorder.DealAmount, buyPrice, sorder.AvgPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v - %.2f(%.2f) btc, + %.2f(%.2f) cny",
        hg.long.name, hg.tradeAmount, lorder.DealAmount, sellPrice, lorder.AvgPrice))
    gmvc.Logger.Println("")

    now := time.Now()
    d := time.Unix(now.Unix() - hg.started.Unix(), 0)
    gmvc.Logger.Println(fmt.Sprintf("profit: %.4f btc, %.2f(%.2f) cny, %v min, %v round %v",
        hg.btc, hg.tcny * hg.tradeAmount, hg.cny * hg.tradeAmount,
        d.Format("15:04:05"), hg.tradeNum, now.Format("15:04:05")))
    gmvc.Logger.Println("")
}

func (hg *Hedger) closeShort(price float64) int64 {
    return hg.short.Buy(hg.tradeAmount * price)
}

func (hg *Hedger) closeLong(price float64) int64 {
    amount := hg.tradeAmount
    if hg.long.lastBuy > 0 {
        amount = hg.long.lastBuy
    }
    return hg.long.Sell(amount)
}


