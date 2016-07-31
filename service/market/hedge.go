package market

import (
    "time"
    "fmt"
    "github.com/roydong/gmvc"
    "log"
    "sync"
)

type Hedge struct {
    zuo, you *Market
    short, long *Market

    tradeAmount float64
    minTradeMargin float64
    marginLevel int
    levelValue float64
    levelEvalAt time.Time
    levelUpNum int

    minAvg, midAvg, maxAvg *averager

    running       bool
    state         int
    started       time.Time
    roundNum      int

    tcny        float64
    cny         float64
    amount      float64
}


func NewHedge(zuo, you *Market) *Hedge {
    hg := &Hedge{
        zuo: zuo,
        you: you,

        minAvg: newAverager(15),
        midAvg: newAverager(300),
        maxAvg: newAverager(15),

        levelValue: 3,
        minTradeMargin: 5,
    }

    conf := gmvc.Store.Tree("config.hedger")
    if v, has := conf.Float("min_trade_margin"); has {
        hg.minTradeMargin = v
    }
    if v, has := conf.Float("margin_level_value"); has {
        hg.levelValue = v
    }
    hg.tradeAmount, _ = conf.Float("trade_amount")

    return hg
}

func (hg *Hedge) Start() {
    hg.zuo.SyncBalance()
    hg.you.SyncBalance()
    amount := hg.zuo.amount + hg.you.amount
    cny := hg.zuo.cny + hg.you.cny
    gmvc.Logger.Println("--------")
    gmvc.Logger.Println(fmt.Sprintf("btc: %.4f, cny: %.2f", amount, cny))

    hg.running = true
    hg.started = time.Now()
    hg.roundNum = 0
    hg.state = StateClose
    hg.marginLevel = 1
    hg.levelEvalAt = hg.started

    go hg.evalMargins(1000 * time.Millisecond)
    go hg.arbitrage(500 * time.Millisecond)
}

func (hg *Hedge) Stop() {
    hg.running = false
    hg.state = StateClose
}

func (hg *Hedge) evalMargins(interval time.Duration) {
    wg := &sync.WaitGroup{}
    var idx int64
    for hg.running {
        time.Sleep(interval)
        idx++
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

        margin := youTicker.Last - zuoTicker.Last
        if hg.midAvg.Len() > 0 {
            if margin <= hg.midAvg.Avg() - hg.tradeMargin() {
                hg.minAvg.AddPeek(false, idx, margin)
            } else if margin >= hg.midAvg.Avg() + hg.tradeMargin() {
                hg.maxAvg.AddPeek(true, idx, margin)
            }
        }
        if overflow, i := hg.midAvg.Add(idx, margin); overflow {
            hg.minAvg.CutTail(i)
            hg.maxAvg.CutTail(i)
        }

        hg.evalLevel()

        log.Println(fmt.Sprintf("%.3f(%v) <= %.3f(%.3f, %v) => %.3f(%v)",
                                hg.minAvgMargin(), hg.minAvg.Len(),
                                hg.midAvg.Avg(), margin, hg.marginLevel,
                                hg.maxAvgMargin(), hg.maxAvg.Len()))
    }
}

func (hg *Hedge) evalLevel() {
    var max, min int64
    if hg.minAvg.Len() > 0 {
        min = int64((hg.midAvg.Avg() - hg.minAvg.Avg() - hg.minTradeMargin) / hg.levelValue)
    }
    if hg.maxAvg.Len() > 0 {
        max = int64((hg.maxAvg.Avg() - hg.midAvg.Avg() - hg.minTradeMargin) / hg.levelValue)
    }

    level := int(gmvc.Max(min, max))
    now := time.Now()
    if level >=0 && level < hg.marginLevel {
        if now.Unix() > hg.levelEvalAt.Unix() + 1800  {
            hg.marginLevel--
            hg.levelEvalAt = now
        }
    } else if level > hg.marginLevel {
        hg.levelUpNum++
        if (hg.minAvg.Len() > 1 || hg.maxAvg.Len() > 1) {
            hg.marginLevel = level
            hg.levelUpNum = 0
            hg.levelEvalAt = now
        }
    }
}

func (hg *Hedge) minAvgMargin() float64 {
    if hg.minAvg.Len() > 1 {
        return hg.minAvg.Avg()
    }
    return hg.midAvg.Avg() - hg.tradeMargin()
}

func (hg *Hedge) maxAvgMargin() float64 {
    if hg.maxAvg.Len() > 1 {
        return hg.maxAvg.Avg()
    }
    return hg.midAvg.Avg() + hg.tradeMargin()
}

func (hg *Hedge) tradeMargin() float64 {
    return hg.minTradeMargin + hg.levelValue * float64(hg.marginLevel)
}

func (hg *Hedge) arbitrage(interval time.Duration) {
    wg := &sync.WaitGroup{}
    for hg.running {
        time.Sleep(interval)
        if hg.midAvg.Len() < 50 {
            continue
        }

        wg.Add(2)
        go func() {
            hg.zuo.UpdateDepth()
            wg.Done()
        }()
        go func() {
            hg.you.UpdateDepth()
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

        zuoBuyPrice := hg.zuo.GetBuyPrice(hg.tradeAmount)
        zuoSellPrice := hg.zuo.GetSellPrice(hg.tradeAmount)

        youBuyPrice := hg.you.GetBuyPrice(hg.tradeAmount)
        youSellPrice := hg.you.GetSellPrice(hg.tradeAmount)

        var margin float64
        if hg.state == StateClose {

            //尝试判断是否可以右手做空(左手多), 以右手的最近买单价 - 左手的卖单价(margin)和(min max avg)相关参数比较
            margin = youSellPrice - zuoBuyPrice

            //满足最小差价条件,并且超过最大差价
            if margin >= hg.maxAvgMargin() {
                gmvc.Logger.Println(fmt.Sprintf("open positoin(youSell - zuoBuy %.2f):", margin))
                hg.openPosition(hg.you, youSellPrice, hg.zuo, zuoBuyPrice)
                continue
            }

            //尝试判断是否可以左手做空(右手多), 以右手的最近卖单价 - 左手的买单价(margin)和(min max avg)相关参数比较
            margin = youBuyPrice - zuoSellPrice

            //满足最小差价条件,并且低于最小差价
            if margin <= hg.minAvgMargin() {
                gmvc.Logger.Println(fmt.Sprintf("open position(youBuy - zuoSell %.2f):", margin))
                hg.openPosition(hg.zuo, zuoSellPrice, hg.you, youBuyPrice)
                continue
            }

        } else {

            //如果是右手做空
            if (hg.short.name == hg.you.name) {
                margin = youBuyPrice - zuoSellPrice

                //差价低于平均差价即可平仓
                if margin <= hg.midAvg.Avg() {
                    gmvc.Logger.Println(fmt.Sprintf("close position(youBuy - zuoSell %.2f):", margin))
                    hg.closePosition(youBuyPrice, zuoSellPrice)
                }

            //如果是左手做空的
            } else {
                margin = youSellPrice - zuoBuyPrice

                //差价高于平均差价即可平仓
                if margin >= hg.midAvg.Avg() {
                    gmvc.Logger.Println(fmt.Sprintf("close position(youSell - zuoBuy %.2f):", margin))
                    hg.closePosition(zuoBuyPrice, youSellPrice)
                }
            }
        }
    }
}

func (hg *Hedge) openPosition(short *Market, shortSellPrice float64, long *Market, longBuyPrice float64) {
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
            short.amountChange -= sorder.DealAmount
            short.cnyChange += sorder.AvgPrice * sorder.DealAmount
            hg.cny += sorder.AvgPrice
            break
        }
    }
    for _ = range time.Tick(500 * time.Millisecond) {
        lorder = long.OrderInfo(lid)
        if lorder.Status == 2 {
            long.amountChange += lorder.DealAmount
            long.cnyChange -= lorder.AvgPrice * lorder.DealAmount
            hg.cny -= lorder.AvgPrice
            break
        }
    }

    gmvc.Logger.Println(fmt.Sprintf("   short: %v - %.4f(%.4f) btc + %.2f(%.2f) cny",
        short.name, hg.tradeAmount, sorder.DealAmount, shortSellPrice, sorder.AvgPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v + %.4f(%.4f) btc - %.2f(%.2f) cny",
        long.name, hg.tradeAmount, lorder.DealAmount, longBuyPrice, lorder.AvgPrice))
    gmvc.Logger.Println("")
}

func (hg *Hedge) openShort(short *Market, sellPrice float64) int64 {
    id := short.Sell(hg.tradeAmount + short.amountChange)
    hg.short = short
    return id
}

func (hg *Hedge) openLong(long *Market, buyPrice float64) int64 {
    id := long.Buy(hg.tradeAmount * buyPrice)
    hg.long = long
    return id
}


func (hg *Hedge) closePosition(buyPrice, sellPrice float64) {
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
    hg.roundNum++

    //交易统计
    var sorder, lorder Order
    for _ = range time.Tick(500 * time.Millisecond) {
        sorder = hg.short.OrderInfo(sid)
        if sorder.Status == 2 {
            hg.short.amountChange += sorder.DealAmount
            hg.short.cnyChange -= sorder.AvgPrice * sorder.DealAmount
            hg.cny -= sorder.AvgPrice
            break
        }
    }
    for _ = range time.Tick(500 * time.Millisecond) {
        lorder = hg.long.OrderInfo(lid)
        if lorder.Status == 2 {
            hg.long.amountChange -= lorder.DealAmount
            hg.long.cnyChange += lorder.AvgPrice * lorder.DealAmount
            hg.cny += lorder.AvgPrice
            break
        }
    }

    gmvc.Logger.Println(fmt.Sprintf("   short: %v + %.4f(%.4f) btc - %.2f(%.2f) cny",
        hg.short.name, hg.tradeAmount, sorder.DealAmount, buyPrice, sorder.AvgPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v - %.4f(%.4f) btc + %.2f(%.2f) cny",
        hg.long.name, hg.tradeAmount, lorder.DealAmount, sellPrice, lorder.AvgPrice))
    gmvc.Logger.Println("")

    now := time.Now()
    d := time.Unix(now.Unix() - hg.started.Unix() - 28800, 0)
    gmvc.Logger.Println(fmt.Sprintf("result: %.4f btc, %.2f(%.2f) cny, %v/%v",
                                    hg.long.amountChange + hg.short.amountChange,
                                    hg.long.cnyChange + hg.short.cnyChange, hg.cny * hg.tradeAmount,
                                    hg.roundNum, d.Format("15:04:05")))
    gmvc.Logger.Println("")
}

func (hg *Hedge) closeShort(price float64) int64 {
    return hg.short.Buy(hg.tradeAmount * price)
}

func (hg *Hedge) closeLong(price float64) int64 {
    return hg.long.Sell(hg.tradeAmount)
}


