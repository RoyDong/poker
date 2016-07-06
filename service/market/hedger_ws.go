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
    OrderStatusCreated = 0
    OrderStatusPartial = 1
    OrderStatusComplete = 2
    OrderStatusCancel = -1
    OrderStatusCanceling = 4

    StateOpen  = iota
    StateOpenPending
    StateClose
    StateClosePending
)

type HedgerWS struct {
    zuo *OKFutureWS
    you *OKFutureWS

    short *OKFutureWS
    long *OKFutureWS

    depthUpdated chan int64
    lastOrder chan Order
    currentOrders map[int64]Order

    tradeAmount float64
    tradePriceOffset float64
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
    pending       bool

    started       time.Time
    tradeNum      int

    tcny        float64
    cny         float64
    btc         float64
}


func NewHedgerWS(zuo, you *OKFutureWS) *HedgerWS {
    hg := &HedgerWS{
        zuo: zuo,
        you: you,

        depthUpdated: make(chan int64, 1),
        lastOrder: make(chan int64, 1),
        currentOrders: make(map[int64]Order, 10),

        minMargin: math.Inf(1),
        maxMargin: math.Inf(-1),

        tickerNum: 200,
        margins: make(map[int64]float64),
        marginList: list.New(),

        state: StateClose,
        stoped: false,
        pending: false,
    }

    conf := gmvc.Store.Tree("config.hedger")
    hg.minTradeMargin, _ = conf.Float("min_trade_margin")
    hg.tradeAmount, _ = conf.Float("trade_amount")
    hg.tradePriceOffset, _ = conf.Float("trade_price_offset")

    hg.zuo.AddHandler("depth", hg.syncDepthUpdated)
    hg.you.AddHandler("depth", hg.syncDepthUpdated)
    hg.zuo.AddHandler("order", hg.syncOrder)
    hg.you.AddHandler("order", hg.syncOrder)

    return hg
}

func (hg *HedgerWS) Start() {
    hg.stoped = false
    hg.tradeNum = 0
    hg.started = time.Now()

    go hg.updateMargins(1 * time.Second)
    go hg.arbitrage()
    go hg.checkPending()

    gmvc.Logger.Println("started...")
}

func (hg *HedgerWS) syncOrder(args ...interface{}) {
    order, _ := args[0].(Order)
    hg.currentOrders[order.Id] = order
    hg.lastOrder <-order
}

func (hg *HedgerWS) syncDepthUpdated(args ...interface{}) {
    updated, _ := args[2].(int64)
    if len(hg.depthUpdated) > 0 {
        <-hg.depthUpdated
    }
    hg.depthUpdated <-updated
}

func (hg *HedgerWS) Stop() {
    hg.stoped = true
}

func (hg *HedgerWS) updateMargins(interval time.Duration) {
    var idx int64
    for _ = range time.Tick(interval) {
        if hg.stoped {
            return
        }
        zuoTicker := hg.zuo.LastTicker()
        youTicker := hg.you.LastTicker()
        if zuoTicker.Time == 0 || youTicker.Time == 0 {
            continue
        }
        maxId := gmvc.Max(zuoTicker.Time, youTicker.Time)
        if idx >= maxId {
            continue
        }
        idx = maxId
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
            hg.minMargin - hg.avgMargin, hg.avgMargin, margin, hg.maxMargin - hg.avgMargin))
    }
}

func (hg *HedgerWS) getMinMargin() (int64, float64) {
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

func (hg *HedgerWS) getMaxMargin() (int64, float64) {
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


func (hg *HedgerWS) arbitrage() {
    for !hg.stoped {
        if hg.marginList.Len() < 10 {
            continue
        }
        if len(hg.zuo.lastAsks) == 0 {
            gmvc.Logger.Println(hg.zuo.Name() + " depth is empty")
            continue
        }
        if len(hg.you.lastAsks) == 0 {
            gmvc.Logger.Println(hg.you.Name() + " depth is empty")
            continue
        }

        /*
        zuoBuyPrice := GetBuyPrice(hg.tradeAmount, hg.zuo.lastAsks)
        zuoSellPrice := GetSellPrice(hg.tradeAmount, hg.zuo.lastBids)
        youBuyPrice := GetBuyPrice(hg.tradeAmount, hg.you.lastAsks)
        youSellPrice := GetSellPrice(hg.tradeAmount, hg.you.lastBids)
        */

        zuoPrice := hg.zuo.lastTicker.Last
        youPrice := hg.you.lastTicker.Last
        margin := youPrice - zuoPrice
        if hg.state == StateClose {

            //尝试判断是否可以右手做空(左手多), 以右手的最近买单价 - 左手的卖单价(margin)和(min max avg)相关参数比较
            //margin = youSellPrice - zuoBuyPrice

            //满足最小差价条件,并且超过最大差价
            if margin - hg.avgMargin >= hg.minTradeMargin && margin >= hg.maxMargin {
                gmvc.Logger.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))
                //hg.openPosition(hg.you, youSellPrice, hg.zuo, zuoBuyPrice)
                hg.state = StateOpenPending
                hg.short = hg.you
                go hg.you.OpenPosition(TypeOpenShort, hg.tradeAmount, youPrice - hg.tradePriceOffset)
                hg.long = hg.zuo
                go hg.zuo.OpenPosition(TypeOpenLong, hg.tradeAmount, zuoPrice + hg.tradePriceOffset)
                continue
            }

            //尝试判断是否可以左手做空(右手多), 以右手的最近卖单价 - 左手的买单价(margin)和(min max avg)相关参数比较
            //margin = youBuyPrice - zuoSellPrice

            //满足最小差价条件,并且低于最小差价
            if hg.avgMargin - margin >= hg.minTradeMargin && margin <= hg.minMargin {
                gmvc.Logger.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))
                //hg.openPosition(hg.zuo, zuoSellPrice, hg.you, youBuyPrice)
                hg.state = StateOpenPending
                hg.short = hg.zuo
                hg.long = hg.you
                go hg.zuo.OpenPosition(TypeOpenShort, hg.tradeAmount, zuoPrice - hg.tradePriceOffset)
                go hg.you.OpenPosition(TypeOpenLong, hg.tradeAmount, youPrice + hg.tradePriceOffset)
                continue
            }

        } else if hg.state == StateOpen {

            //如果是右手做空
            if (hg.short.Name() == hg.you.Name()) {
                //margin = youBuyPrice - zuoSellPrice

                //差价低于平均差价即可平仓
                if margin <= hg.avgMargin {
                    gmvc.Logger.Println(fmt.Sprintf("youBuy - zuoSell %.2f", margin))
                    //hg.closePosition(youBuyPrice, zuoSellPrice)

                    hg.state = StateClosePending
                    go hg.short.OpenPosition(TypeCloseShort, hg.tradeAmount, youPrice + hg.tradePriceOffset)
                    go hg.long.OpenPosition(TypeCloseLong, hg.tradeAmount, zuoPrice - hg.tradePriceOffset)
                }

            //如果是左手做空的
            } else {
                //margin = youSellPrice - zuoBuyPrice

                //差价高于平均差价即可平仓
                if margin >= hg.avgMargin {
                    gmvc.Logger.Println(fmt.Sprintf("youSell - zuoBuy %.2f", margin))
                    //hg.closePosition(zuoBuyPrice, youSellPrice)

                    hg.state = StateClosePending
                    go hg.short.OpenPosition(TypeCloseShort, hg.tradeAmount, zuoPrice + hg.tradePriceOffset)
                    go hg.long.OpenPosition(TypeCloseLong, hg.tradeAmount, youPrice - hg.tradePriceOffset)
                }
            }
        }
    }
}

func (hg *HedgerWS) pending() {
    for !hg.stoped {
        <-hg.lastOrder
        if hg.state == StateOpenPending {
            for order := range hg.currentOrders {

            }
        } else if hg.state == StateClosePending {
            for order := range hg.currentOrders {

            }
        }
    }
}

func (hg *HedgerWS) checkPending() {
    for _ = range time.Tick(5 * time.Second) {
        if !hg.stoped {
            return
        }
        if hg.state == StateOpenPending {
            if !hg.pending {
                //cancel orders
                hg.state = StateClose
            }
        } else if hg.state == StateClosePending {
            if !hg.pending {
                //cancel orders
                hg.state = StateOpen
            }
        }
    }
}

func (hg *HedgerWS) openPosition(short *OKFutureWS, shortSellPrice float64, long *OKFutureWS, longBuyPrice float64) {



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

func (hg *HedgerWS) openShort(short *OKFutureWS, sellPrice float64) int64 {
    id := short.Sell(hg.tradeAmount)
    hg.short = short
    return id
}

func (hg *HedgerWS) openLong(long *OKFutureWS, buyPrice float64) int64 {
    delta := 0.0;
    if long.name == "okcoin" {
        delta = 0.006
    }
    id := long.Buy((hg.tradeAmount + delta) * buyPrice)
    hg.long = long
    return id
}


func (hg *HedgerWS) closePosition(buyPrice, sellPrice float64) {
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

func (hg *HedgerWS) closeShort(price float64) int64 {
    delta := 0.0;
    if hg.short.name == "okcoin" {
        delta = 0.006
    }
    return hg.short.Buy((hg.tradeAmount + delta) * price)
}

func (hg *HedgerWS) closeLong(price float64) int64 {
    return hg.long.Sell(hg.tradeAmount)
}



