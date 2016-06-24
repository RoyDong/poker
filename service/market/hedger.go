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

type marketPosition struct {

}

type Hedger struct {
    zuo *Market
    you *Market

    short *Market
    long *Market

    tradeAmount float64
    minTradeMargin float64

    zuoProfit float64
    youProfit float64
    totalProfit float64



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

    timeDelta     int64
}


func NewHedger(zuo, you *Market) *Hedger {
    hg := &Hedger{
        zuo: zuo,
        you: you,

        tradeAmount: 0.1,
        minTradeMargin: 3,

        minMargin: math.Inf(1),
        maxMargin: math.Inf(-1),

        tickerNum: 60,
        margins: make(map[int64]float64),
        marginList: list.New(),

        state: STATE_CLOSE,
    }

    zuo.SyncBalance()
    you.SyncBalance()

    return hg
}

func (hg *Hedger) Start() {
    hg.stoped = false

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
            idx, _ := el.Value.(int64)
            hg.totalMargin -= hg.margins[idx]
            delete(hg.margins, idx)

            if idx == hg.minMarginTime {
                hg.minMarginTime, hg.minMargin = hg.getMinMargin()
            }

            if idx == hg.maxMarginTime {
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
    for now := range time.Tick(1 * time.Second) {
        if hg.marginList.Len() < 20 {
            continue
        }

        zuoTicker := hg.zuo.FrontTicker()
        youTicker := hg.you.FrontTicker()

        if math.Abs(float64(zuoTicker.Time - now.Unix())) > 3 || math.Abs(float64(youTicker.Time - now.Unix())) > 3 {
            return
        }

        margin := youTicker.Last - zuoTicker.Last

        if hg.state == STATE_CLOSE {
            if margin - hg.avgMargin >= hg.minTradeMargin && hg.maxMargin - margin <= hg.minTradeMargin / 10 {
                hg.openPosition(hg.you, hg.zuo)
            }

            if hg.avgMargin - margin >= hg.minTradeMargin && margin - hg.minMargin <= hg.minTradeMargin / 10 {
                hg.openPosition(hg.zuo, hg.you)
            }
        } else if math.Abs(margin - hg.avgMargin) <= hg.minTradeMargin / 10 {
            hg.closePosition()
        }

    }
}


func (hg *Hedger) openPosition(short, long *Market) {
    var cny float64

    //TODO
    short.Sell(hg.tradeAmount)
    cny = hg.tradeAmount * short.FrontTicker().Last

    gmvc.Logger.Println("open position:")
    gmvc.Logger.Println(fmt.Sprintf("   short: %v sell %.2f btc, + %.2f cny", short.name, hg.tradeAmount, cny))


    cny = hg.tradeAmount * long.FrontTicker().Last
    long.Buy(cny)

    gmvc.Logger.Println(fmt.Sprintf("   long: %v buy %.2f btc, - %.2f cny", long.name, hg.tradeAmount, cny))
    gmvc.Logger.Println("")

    hg.short = short
    hg.long = long

    hg.state = STATE_OPEN

}


func (hg *Hedger) closePosition() {
    var cny float64

    cny = hg.tradeAmount * hg.short.FrontTicker().Last
    hg.short.Buy(cny)

    gmvc.Logger.Println("close position:")
    gmvc.Logger.Println(fmt.Sprintf("   short: %v buy %.2f btc, - %.2f cny", hg.short.name, hg.tradeAmount, cny))

    hg.long.Sell(hg.tradeAmount)
    cny = hg.tradeAmount * hg.long.FrontTicker().Last
    gmvc.Logger.Println(fmt.Sprintf("   long: %v sell %.2f btc, + %.2f cny", hg.long.name, hg.tradeAmount, cny))
    gmvc.Logger.Println("")

    time.Sleep(1 * time.Second)

    hg.zuo.SyncBalance()
    hg.you.SyncBalance()

    total := hg.zuo.cny + hg.you.cny + hg.zuo.btc * hg.zuo.FrontTicker().Last + hg.you.btc * hg.you.FrontTicker().Last
    btc := hg.zuo.btc + hg.you.btc
    cny  = hg.zuo.cny + hg.you.cny

    gmvc.Logger.Println("balance at " +  time.Now().Format("15:04:05"))
    gmvc.Logger.Println(
        fmt.Sprintf("   Total %.2f￥(%.2fB, %.2f￥), %v(%.2fB, %.2f￥), %v(%.2fB, %.2f￥)",
        total, btc, cny, hg.zuo.name, hg.zuo.btc, hg.zuo.cny, hg.you.name, hg.you.btc, hg.you.cny))
    gmvc.Logger.Println("")

    hg.state = STATE_CLOSE
}




