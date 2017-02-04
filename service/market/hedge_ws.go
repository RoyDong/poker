package market

import (
    "time"
    "container/list"
    "math"
    "fmt"
    "github.com/roydong/gmvc"
    "log"
    "sync"
)


const (
    OrderStatusCreated   = 0
    OrderStatusPartial   = 1
    OrderStatusComplete  = 2
    OrderStatusCancel    = -1
    OrderStatusCanceling = 4

)

type HedgeWS struct {
    zuo *OKFutureWS
    you *OKFutureWS

    short *OKFutureWS
    long *OKFutureWS

    shortAmount, longAmount float64
    shortPrice, longPrice float64
    shortOrderId, longOrderId int64
    tradeMargin float64

    direction int

    trade chan Trade
    lastTrade chan Trade
    lastOrder chan Order

    tradeAmount float64
    realTradeAmount float64
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

    state         int
    pendingTime   int64

    started       time.Time
    tradeNum      int

    tcny        float64
    cny         float64
    btc         float64

    wg *sync.WaitGroup
}


func NewHedgeWS(zuo, you *OKFutureWS) *HedgeWS {
    hg := &HedgeWS{
        zuo: zuo,
        you: you,

        trade: make(chan Trade, 5),
        lastTrade: make(chan Trade, 5),
        lastOrder: make(chan Order, 10),

        minMargin: math.Inf(1),
        maxMargin: math.Inf(-1),

        tickerNum: 200,
        margins: make(map[int64]float64),
        marginList: list.New(),

        state: StateStop,
        wg: &sync.WaitGroup{},
    }

    conf := gmvc.Store.Tree("config.hedger")
    hg.tradeAmount,      _ = conf.Float("trade_amount")
    hg.minTradeMargin,   _ = conf.Float("min_trade_margin")

    hg.zuo.AddHandler("new_trade", hg.syncTrade)
    hg.you.AddHandler("new_trade", hg.syncTrade)
    hg.zuo.AddHandler("last_trade", hg.syncLastTrade)
    hg.you.AddHandler("last_trade", hg.syncLastTrade)

    return hg
}

func (hg *HedgeWS) Start() {
    hg.btc, _ = hg.zuo.GetBalance()
    hg.btc, _ = hg.you.GetBalance()

    hg.state = StateClose
    hg.tradeNum = 0
    hg.started = time.Now()

    go hg.updateMargins()
    go hg.arbitrage()

    gmvc.Logger.Println("started...")
}

func (hg *HedgeWS) syncTrade(args ...interface{}) {
    trade, _ := args[0].(Trade)
    if len(hg.trade) > cap(hg.trade) {
        <-hg.trade
    }
    hg.trade <-trade
}

func (hg *HedgeWS) syncLastTrade(args ...interface{}) {
    trade, _ := args[0].(Trade)
    if len(hg.lastTrade) > cap(hg.lastTrade) {
        <-hg.lastTrade
    }
    hg.lastTrade <-trade
}

func (hg *HedgeWS) Stop() {
    hg.state = StateStop
}

func (hg *HedgeWS) updateMargins() {
    for hg.state != StateStop {
        trade := <-hg.trade
        idx := trade.No
        zuoPrice := hg.zuo.lastTrade.Price
        youPrice := hg.you.lastTrade.Price
        if zuoPrice <= 0 || youPrice <= 0 {
            continue
        }

        margin := youPrice - zuoPrice
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

        hg.avgMargin = hg.totalMargin / float64(hg.marginList.Len())
        if hg.minMargin > margin {
            hg.minMargin = margin
            hg.minMarginTime = idx
        }
        if hg.maxMargin < margin {
            hg.maxMargin = margin
            hg.maxMarginTime = idx
        }

        log.Println(fmt.Sprintf("%.3f <= %.3f(%.3f) => %.3f",
            hg.minMargin - hg.avgMargin, hg.avgMargin, margin - hg.avgMargin, hg.maxMargin - hg.avgMargin))
    }
}

func (hg *HedgeWS) getMinMargin() (int64, float64) {
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

func (hg *HedgeWS) getMaxMargin() (int64, float64) {
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


func (hg *HedgeWS) arbitrage() {
    for hg.state != StateStop {
        <-hg.lastTrade
        if hg.marginList.Len() < 100 {
            continue
        }
        zuoPrice := hg.zuo.lastTrade.Price
        youPrice := hg.you.lastTrade.Price
        margin := youPrice - zuoPrice
        log.Println(margin, hg.state)

        switch hg.state {
        case StateClose:
            //满足最小差价条件,并且超过最大差价,右手空，左手多 direction = 1
            if margin - hg.avgMargin >= hg.minTradeMargin && margin >= hg.maxMargin {
                hg.direction = 1
                hg.openPosition(hg.you, hg.zuo, youPrice, zuoPrice, margin)
                continue
            }
            //满足最小差价条件,并且低于最小差价,左手空，右手多 direction = -1
            if hg.avgMargin - margin >= hg.minTradeMargin && margin <= hg.minMargin {
                hg.direction = -1
                hg.openPosition(hg.zuo, hg.you, zuoPrice, youPrice, margin)
                continue
            }

        case StateOpen:
            //如果是右手做空
            if (hg.direction == 1) {
                //差价低于平均差价即可平仓
                if margin <= hg.avgMargin {
                    hg.closePosition(youPrice, zuoPrice, margin)
                }
                //如果是左手做空的
            } else {
                //差价高于平均差价即可平仓
                if margin >= hg.avgMargin {
                    hg.closePosition(zuoPrice, youPrice, margin)
                }
            }
        }
    }
}

func (hg *HedgeWS) getCurrentMargin() float64 {
    return hg.you.lastTrade.Price - hg.zuo.lastTrade.Price
}

func (hg *HedgeWS) openPosition(short, long *OKFutureWS, shortPrice, longPrice, margin float64) {
    hg.short = short
    hg.long = long
    hg.tradeMargin = margin
    hg.state = StateOpen
    hg.pendingTime = time.Now().Unix()
    hg.wg.Add(2)
    go func() {
        hg.shortAmount, hg.shortPrice = hg.short.FTrade(TypeOpenShort, hg.tradeAmount, hg.minTradeMargin / 2)
        hg.wg.Done()
    }()
    go func() {
        hg.longAmount, hg.longPrice = hg.long.FTrade(TypeOpenLong, hg.tradeAmount, hg.minTradeMargin / 2)
        hg.wg.Done()
    }()
    hg.wg.Wait()

    gmvc.Logger.Println("open position:")
    gmvc.Logger.Println(fmt.Sprintf("   short: %v - %.2f(%.2f) btc, + %.2f(%.2f) usd",
        short.Name(), hg.tradeAmount, hg.shortAmount, shortPrice, hg.shortPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v + %.2f(%.2f) btc, - %.2f(%.2f) usd",
        long.Name(), hg.tradeAmount, hg.longAmount, longPrice, hg.longPrice))
    gmvc.Logger.Println("")
}

func (hg *HedgeWS) closePosition(shortPrice, longPrice, margin float64) {
    hg.state = StateClose
    hg.pendingTime = time.Now().Unix()
    hg.wg.Add(2)
    go func() {
        hg.shortAmount, hg.shortPrice = hg.short.FTrade(TypeCloseShort, hg.shortAmount, 0)
        hg.wg.Done()
    }()
    go func() {
        hg.longAmount, hg.longPrice = hg.long.FTrade(TypeCloseLong, hg.longAmount, 0)
        hg.wg.Done()
    }()
    hg.wg.Wait()

    gmvc.Logger.Println("close position:")
    gmvc.Logger.Println(fmt.Sprintf("   short: %v + %.2f(%.2f) btc, - %.2f(%.2f) cny",
        hg.short.Name(), hg.tradeAmount, hg.shortAmount, shortPrice, hg.shortPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v - %.2f(%.2f) btc, + %.2f(%.2f) cny",
        hg.long.Name(), hg.tradeAmount, hg.longAmount, longPrice, hg.longPrice))
    gmvc.Logger.Println("")

    time.Sleep(1 * time.Second)
}

