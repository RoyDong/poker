package market

import (
    "time"
    "container/list"
    "math"
    "fmt"
    "github.com/roydong/gmvc"
    "log"
)

const (
    STATE_OPEN  = 1
    STATE_CLOSE = 2
)

type Hedger struct {
    zuo *Market
    you *Market

    short *Market
    long *Market

    tradeAmount float64
    minTradeMargin float64

    minMargin     float64
    minMarginTime int64

    maxMargin     float64
    maxMarginTime int64

    totalMargin   float64
    avgMargin     float64

    tickerNum     int
    margins       map[int64]float64
    marginList    *list.List

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

        minMargin: math.Inf(1),
        maxMargin: math.Inf(-1),

        tickerNum: 200,
        margins: make(map[int64]float64),
        marginList: list.New(),

        state: STATE_CLOSE,
    }

    zuo.SyncBalance()
    you.SyncBalance()

    conf := gmvc.Store.Tree("config.hedger")
    hg.minTradeMargin, _ = conf.Float64("min_trade_margin")
    hg.tradeAmount, _ = conf.Float64("trade_amount")

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

    go hg.zuo.SyncTicker(1 * time.Second)
    go hg.you.SyncTicker(1 * time.Second)
    go hg.updateMargins(1 * time.Second)
    go hg.arbitrage(1 * time.Second)
}

func (hg *Hedger) Stop() {
    hg.stoped = true
}

func (hg *Hedger) updateMargins(interval time.Duration) {
    for _ = range time.Tick(interval) {
        zuoTicker := hg.zuo.FrontTicker()
        youTicker := hg.you.FrontTicker()
        if zuoTicker == nil || youTicker == nil {
            continue
        }

        idx := zuoTicker.Time
        margin := youTicker.Last - zuoTicker.Last
        hg.totalMargin += margin
        hg.margins[idx] = margin
        hg.marginList.PushFront(idx)
        if hg.marginList.Len() > hg.tickerNum {
            el := hg.marginList.Back()
            hg.marginList.Remove(el)
            i, _ := el.Value.(int64)
            hg.totalMargin -= hg.margins[i]
            delete(hg.margins, i)

            if i == hg.minMarginTime {
                hg.minMarginTime, hg.minMargin = hg.getMinMargin()
            }

            if i == hg.maxMarginTime {
                hg.maxMarginTime, hg.maxMargin = hg.getMaxMargin()
            }

        }

        if hg.minMargin > margin {
            hg.minMargin = margin
            hg.minMarginTime = idx
        }
        if hg.maxMargin < margin {
            hg.maxMargin = margin
            hg.maxMarginTime = idx
        }
        hg.avgMargin = hg.totalMargin / float64(hg.marginList.Len())

        log.Println(fmt.Sprintf("%.2f <= %.2f => %.2f",
            hg.minMargin - hg.avgMargin, hg.avgMargin,  hg.maxMargin - hg.avgMargin))
    }
}

func (hg *Hedger) getMinMargin() (int64, float64) {
    min := math.Inf(1)
    var idx int64
    for el := hg.marginList.Back(); el != nil; el = el.Prev() {
        i, _ := el.Value.(int64)
        v := hg.margins[i]
        if v < min {
            idx = i
            min = v
        }
    }
    return idx, min
}

func (hg *Hedger) getMaxMargin() (int64, float64) {
    max := math.Inf(-1)
    var idx int64
    for el := hg.marginList.Back(); el != nil; el = el.Prev() {
        i, _ := el.Value.(int64)
        v := hg.margins[i]
        if v > max {
            idx = i
            max = v
        }
    }
    return idx, max
}


func (hg *Hedger) arbitrage(interval time.Duration) {
    for _ = range time.Tick(interval) {
        if hg.marginList.Len() < 10 {
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
        if hg.state == STATE_CLOSE {

            //尝试判断是否可以右手做空(左手多), 以右手的最近买单价 - 左手的卖单价(margin)和(min max avg)相关参数比较
            margin = youSellPrice - zuoBuyPrice
            log.Println(fmt.Sprintf("youSell - zuoBuy %.2f(%.2f)", margin, margin - hg.avgMargin))

            //满足最小差价条件,并且超过最大差价
            if margin - hg.avgMargin >= hg.minTradeMargin && margin >= hg.maxMargin {
                //gmvc.Logger.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))
                hg.openPosition(hg.you, youSellPrice, hg.zuo, zuoBuyPrice)
                continue
            }

            //尝试判断是否可以左手做空(右手多), 以右手的最近卖单价 - 左手的买单价(margin)和(min max avg)相关参数比较
            margin = youBuyPrice - zuoSellPrice
            log.Println(fmt.Sprintf("youBuy - zuoSell %.2f(%.2f)", margin, margin - hg.avgMargin))

            //满足最小差价条件,并且低于最小差价
            if hg.avgMargin - margin >= hg.minTradeMargin && margin <= hg.minMargin {
                //gmvc.Logger.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))
                hg.openPosition(hg.zuo, zuoSellPrice, hg.you, youBuyPrice)
                continue
            }

        } else {

            //如果是右手做空
            if (hg.short.name == hg.you.name) {
                margin = youBuyPrice - zuoSellPrice
                log.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))

                //差价低于平均差价即可平仓
                if margin <= hg.avgMargin {
                    //gmvc.Logger.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))
                    hg.closePosition(youBuyPrice, zuoSellPrice)
                }

            //如果是左手做空的
            } else {
                margin = youSellPrice - zuoBuyPrice
                log.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))

                //差价高于平均差价即可平仓
                if margin >= hg.avgMargin {
                    //gmvc.Logger.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))
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

    hg.state = STATE_OPEN

    //交易统计
    time.Sleep(2 * time.Second)
    sorder := short.OrderInfo(sid)
    if sorder == nil {
        return
    }
    if sorder.DealAmount <= 0 {
        sorder.DealAmount = hg.tradeAmount
    }
    if sorder.AvgPrice <= 0 {
        sorder.AvgPrice = shortSellPrice
    }

    lorder := long.OrderInfo(lid)
    if lorder == nil {
        return
    }
    if lorder.DealAmount <= 0 {
        lorder.DealAmount = hg.tradeAmount
    }
    if lorder.AvgPrice <= 0 {
        lorder.AvgPrice = longBuyPrice
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
    id := short.Sell(hg.tradeAmount)
    hg.short = short
    return id
}

func (hg *Hedger) openLong(long *Market, buyPrice float64) int64 {
    delta := 0.0;
    if long.name == "okcoin" {
        delta = 0.006
    }
    id := long.Buy((hg.tradeAmount + delta) * buyPrice)
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
    hg.state = STATE_CLOSE
    hg.tradeNum++

    //交易统计
    time.Sleep(2 * time.Second)
    sorder := hg.short.OrderInfo(sid)
    if sorder == nil {
        return
    }

    if sorder.DealAmount <= 0 {
        sorder.DealAmount = hg.tradeAmount
    }
    if sorder.AvgPrice <= 0 {
        sorder.AvgPrice = buyPrice
    }

    lorder := hg.long.OrderInfo(lid)
    if lorder == nil {
        return
    }

    if lorder.DealAmount <= 0 {
        lorder.DealAmount = hg.tradeAmount
    }
    if lorder.AvgPrice <= 0 {
        lorder.AvgPrice = sellPrice
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
    delta := 0.0;
    if hg.short.name == "okcoin" {
        delta = 0.006
    }
    return hg.short.Buy((hg.tradeAmount + delta) * price)
}

func (hg *Hedger) closeLong(price float64) int64 {
    return hg.long.Sell(hg.tradeAmount)
}


