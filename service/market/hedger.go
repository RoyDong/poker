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

    cny         float64
    btc         float64
    test        bool
}


func NewHedger(zuo, you *Market) *Hedger {
    hg := &Hedger{
        zuo: zuo,
        you: you,

        minMargin: math.Inf(1),
        maxMargin: math.Inf(-1),

        tickerNum: 60,
        margins: make(map[int64]float64),
        marginList: list.New(),

        state: STATE_CLOSE,
    }

    zuo.SyncBalance()
    you.SyncBalance()

    conf := gmvc.Store.Tree("config.hedger")
    hg.minTradeMargin, _ = conf.Float64("min_trade_margin")
    hg.tradeAmount, _ = conf.Float64("trade_amount")
    t, _ := conf.Int("test")
    if t > 0 {
        hg.test = true
    } else {
        hg.test = false
    }

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
    go hg.updateMargins()
    go hg.arbitrage()
}

func (hg *Hedger) Stop() {
    hg.stoped = true
}

func (hg *Hedger) updateMargins() {
    for now := range time.Tick(1 * time.Second) {
        idx := now.Unix() - 5

        zuoTicker := hg.zuo.TickerByTime(idx)
        youTicker := hg.you.TickerByTime(idx)

        if zuoTicker == nil || youTicker == nil {
            continue
        }

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

        log.Println(fmt.Sprintf("minMargin: %.2f, maxMargin: %.2f, avgMargin: %.2f, lastMargin: %.2f",
                                        hg.minMargin, hg.maxMargin, hg.avgMargin, margin))
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


func (hg *Hedger) arbitrage() {
    for _ = range time.Tick(1 * time.Second) {
        if hg.marginList.Len() < 10 {
            continue
        }

        hg.zuo.UpdateDepth()
        hg.you.UpdateDepth()

        if len(hg.zuo.lastAsks) == 0 || len(hg.you.lastAsks) == 0 {
            continue
        }

        zuoBuyPrice := hg.zuo.GetBuyPrice(hg.tradeAmount)
        zuoSellPrice := hg.zuo.GetSellPrice(hg.tradeAmount)

        youBuyPrice := hg.you.GetBuyPrice(hg.tradeAmount)
        youSellPrice := hg.you.GetSellPrice(hg.tradeAmount)

        var margin float64
        var tradeMarginGap = hg.minTradeMargin / 10

        if hg.state == STATE_CLOSE {

            //尝试判断是否可以右手做空(左手多), 以右手的最近买单价 - 左手的卖单价(margin)和(min max avg)相关参数比较
            margin = youSellPrice - zuoBuyPrice
            log.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))

            //满足最小差价条件,并且超过最大差价
            if margin - hg.avgMargin >= hg.minTradeMargin && margin >= hg.maxMargin - tradeMarginGap {
                gmvc.Logger.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))
                hg.openPosition(hg.you, youSellPrice, hg.zuo, zuoBuyPrice)
                continue
            }

            //尝试判断是否可以左手做空(右手多), 以右手的最近卖单价 - 左手的买单价(margin)和(min max avg)相关参数比较
            margin = youBuyPrice - zuoSellPrice
            log.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))

            //满足最小差价条件,并且低于最小差价
            if hg.avgMargin - margin >= hg.minTradeMargin && margin <= hg.minMargin + tradeMarginGap {
                gmvc.Logger.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))
                hg.openPosition(hg.zuo, zuoSellPrice, hg.you, youBuyPrice)
                continue
            }

        } else {

            //如果是右手做空
            if (hg.short.name == hg.you.name) {
                margin = youBuyPrice - zuoSellPrice
                log.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))

                //差价低于平均差价即可平仓
                if margin <= hg.avgMargin + tradeMarginGap {
                    gmvc.Logger.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))
                    hg.closePosition(youBuyPrice, zuoSellPrice)
                }

            //如果是左手做空的
            } else {
                margin = youSellPrice - zuoBuyPrice
                log.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))

                //差价高于平均差价即可平仓
                if margin >= hg.avgMargin - tradeMarginGap {
                    gmvc.Logger.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))
                    hg.closePosition(zuoBuyPrice, youSellPrice)
                }
            }
        }
    }
}


func (hg *Hedger) openPosition(short *Market, shortSellPrice float64, long *Market, longBuyPrice float64) {
    gmvc.Logger.Println("open position:")

    if short.name == "huobi" {
        err := hg.openShort(short, shortSellPrice)
        if err != nil {
            return
        }

        hg.openLong(long, longBuyPrice)
    } else {
        err := hg.openLong(long, longBuyPrice)
        if err != nil {
            return
        }

        hg.openShort(short, shortSellPrice)
    }

    gmvc.Logger.Println("")
    hg.state = STATE_OPEN
}

func (hg *Hedger) openShort(short *Market, sellPrice float64) error {
    var err error
    if !hg.test {
        err = short.Sell(hg.tradeAmount)
        if err != nil {
            return err
        }
    }
    cny := hg.tradeAmount * sellPrice
    gmvc.Logger.Println(fmt.Sprintf("   short: %v sell %.2f btc, + %.2f cny", short.name, hg.tradeAmount, sellPrice))
    hg.short = short

    hg.btc -= hg.tradeAmount
    hg.cny += cny


    return err
}

func (hg *Hedger) openLong(long *Market, buyPrice float64) error {
    delta := 0.0;

    if long.name == "okcoin" {
        delta = 0.005
    }
    cny := (hg.tradeAmount + delta) * buyPrice

    var err error
    if !hg.test {
        err = long.Buy(cny)
        if err != nil {
            return err
        }
    }
    gmvc.Logger.Println(fmt.Sprintf("   long: %v buy %.2f btc, - %.2f cny", long.name, hg.tradeAmount, buyPrice))
    hg.long = long

    hg.btc += hg.tradeAmount
    hg.cny -= hg.tradeAmount * buyPrice

    return err
}


func (hg *Hedger) closePosition(buyPrice, sellPrice float64) {
    gmvc.Logger.Println("close position:")

    if hg.short.name == "huobi" {
        err := hg.closeShort(buyPrice)
        if err != nil {
            return
        }

        hg.closeLong(sellPrice)
    } else {
        err := hg.closeLong(sellPrice)
        if err != nil {
            return
        }

        hg.closeShort(buyPrice)
    }

    gmvc.Logger.Println("")

    //hg.zuo.SyncBalance()

    hg.tradeNum++
    now := time.Now()
    gmvc.Logger.Println(fmt.Sprintf("info: %v min, %v rounds, %v", (now.Unix() - hg.started.Unix()) / 60, hg.tradeNum, now.Format("15:04:05")))
    gmvc.Logger.Println(
        fmt.Sprintf("   Total(%.4f, %.2f), %v(%.4f, %.2f), %v(%.4f, %.2f)",
        hg.btc, hg.cny, hg.zuo.name, hg.zuo.btc, hg.zuo.cny, hg.you.name, hg.you.btc, hg.you.cny))
    gmvc.Logger.Println("")

    hg.state = STATE_CLOSE
}

func (hg *Hedger) closeShort(price float64) error {
    delta := 0.0;

    if hg.short.name == "okcoin" {
        delta = 0.005
    }
    cny := (hg.tradeAmount + delta) * price
    var err error
    if !hg.test {
        err = hg.short.Buy(cny)
        if err != nil {
            return err
        }
    }

    gmvc.Logger.Println(fmt.Sprintf("   short: %v buy %.2f btc, - %.2f cny", hg.short.name, hg.tradeAmount, price))

    hg.btc += hg.tradeAmount
    hg.cny -= hg.tradeAmount * price

    return err
}

func (hg *Hedger) closeLong(price float64) error {
    var err error
    if !hg.test {
        err = hg.long.Sell(hg.tradeAmount)
        if err != nil {
            return err
        }
    }
    cny := hg.tradeAmount * price
    gmvc.Logger.Println(fmt.Sprintf("   long: %v sell %.2f btc, + %.2f cny", hg.long.name, hg.tradeAmount, price))

    hg.btc -= hg.tradeAmount
    hg.cny += cny

    return err
}


