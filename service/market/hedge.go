package market

import (
    "time"
    "fmt"
    "github.com/roydong/gmvc"
    "log"
    "sync"
    "math"
)

type Hedge struct {
    zuo, you *Market
    short, long *Market

    tradeAmount float64
    maxTradeAmount float64
    minTradeMargin float64
    marginLevel int
    levelValue float64
    levelEvalAt time.Time
    levelUpNum int

    avg, minAvg, maxAvg *averager

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

        avg: newAverager(300),
        minAvg: newAverager(15),
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

    return hg
}

func (hg *Hedge) Start() {
    hg.zuo.SyncBalance()
    hg.you.SyncBalance()
    hg.maxTradeAmount = math.Min(hg.zuo.amount, hg.you.amount) - 0.01
    amount := hg.zuo.amount + hg.you.amount
    cny := hg.zuo.cny + hg.you.cny
    gmvc.Logger.Println("--------")
    gmvc.Logger.Println(fmt.Sprintf("btc: %.4f, cny: %.2f", amount, cny))

    hg.running = true
    hg.started = time.Now()
    hg.roundNum = 0
    hg.state = StateClose

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
        if hg.avg.Len() > 0 {
            if margin <= hg.avg.Avg() - hg.tradeMargin() {
                hg.minAvg.AddPeek(false, idx, margin)
            } else if margin >= hg.avg.Avg() + hg.tradeMargin() {
                hg.maxAvg.AddPeek(true, idx, margin)
            }
        }
        if overflow, i := hg.avg.Add(idx, margin); overflow {
            hg.minAvg.CutTail(i)
            hg.maxAvg.CutTail(i)
        }

        hg.evalLevel()

        log.Println(fmt.Sprintf("%.2f(%v) <= %.2f(%.2f, %v) => %.2f(%v)",
                                hg.minAvgMargin(), hg.minAvg.Len(),
                                hg.avg.Avg(), margin, hg.marginLevel,
                                hg.maxAvgMargin(), hg.maxAvg.Len()))
    }
}

func (hg *Hedge) evalLevel() {
    var max, min int64
    if hg.minAvg.Len() > 0 {
        min = int64((hg.avg.Avg() - hg.minAvg.Avg() - hg.minTradeMargin) / hg.levelValue)
    }
    if hg.maxAvg.Len() > 0 {
        max = int64((hg.maxAvg.Avg() - hg.avg.Avg() - hg.minTradeMargin) / hg.levelValue)
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
    return hg.avg.Avg() - hg.tradeMargin()
}

func (hg *Hedge) maxAvgMargin() float64 {
    if hg.maxAvg.Len() > 1 {
        return hg.maxAvg.Avg()
    }
    return hg.avg.Avg() + hg.tradeMargin()
}

func (hg *Hedge) tradeMargin() float64 {
    return hg.minTradeMargin + hg.levelValue * float64(hg.marginLevel)
}

func (hg *Hedge) arbitrage(interval time.Duration) {
    wg := &sync.WaitGroup{}
    for hg.running {
        time.Sleep(interval)
        if hg.avg.Len() < 50 {
            continue
        }

        //获取最新盘口深度
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

        zuoBuyPrice, zuoBuyAmount := hg.zuo.GetAskDepth(1)
        zuoSellPrice, zuoSellAmount := hg.zuo.GetBidDepth(1)

        youBuyPrice, youBuyAmount := hg.you.GetAskDepth(1)
        youSellPrice, youSellAmount := hg.you.GetBidDepth(1)

        var margin float64
        if hg.state == StateClose {

            //尝试判断是否可以右手做空(左手多), 以右手的最近买单价 - 左手的卖单价(margin)和(min max avg)相关参数比较
            margin = youSellPrice - zuoBuyPrice

            //满足最小差价条件,并且超过最大差价
            if margin >= hg.maxAvgMargin() {
                gmvc.Logger.Println(fmt.Sprintf("open positoin(youSell - zuoBuy %.2f %v):", margin, hg.marginLevel))
                hg.tradeAmount = math.Min(math.Min(youSellAmount, zuoBuyAmount), hg.maxTradeAmount)
                hg.openPosition(hg.you, youSellPrice, hg.zuo, zuoBuyPrice)
                continue
            }

            //尝试判断是否可以左手做空(右手多), 以右手的最近卖单价 - 左手的买单价(margin)和(min max avg)相关参数比较
            margin = youBuyPrice - zuoSellPrice

            //满足最小差价条件,并且低于最小差价
            if margin <= hg.minAvgMargin() {
                gmvc.Logger.Println(fmt.Sprintf("open position(youBuy - zuoSell %.2f %v):", margin, hg.marginLevel))
                hg.tradeAmount = math.Min(math.Min(youBuyAmount, zuoSellAmount), hg.maxTradeAmount)
                hg.openPosition(hg.zuo, zuoSellPrice, hg.you, youBuyPrice)
                continue
            }

        } else {

            //如果是右手做空
            if (hg.short.name == hg.you.name) {
                margin = youBuyPrice - zuoSellPrice

                //差价低于平均差价即可平仓
                if margin <= hg.avg.Avg() {
                    gmvc.Logger.Println(fmt.Sprintf("close position(youBuy - zuoSell %.2f):", margin))
                    hg.closePosition(youBuyPrice, zuoSellPrice)
                }

            //如果是左手做空的
            } else {
                margin = youSellPrice - zuoBuyPrice

                //差价高于平均差价即可平仓
                if margin >= hg.avg.Avg() {
                    gmvc.Logger.Println(fmt.Sprintf("close position(youSell - zuoBuy %.2f):", margin))
                    hg.closePosition(zuoBuyPrice, youSellPrice)
                }
            }
        }
    }
}

func (hg *Hedge) openPosition(short *Market, shortSellPrice float64, long *Market, longBuyPrice float64) {
    wg := &sync.WaitGroup{}
    wg.Add(2)
    var sorder, lorder Order
    go func() {
        sorder = hg.openShort(short, shortSellPrice)
        wg.Done()
    }()
    go func() {
        lorder = hg.openLong(long, longBuyPrice)
        wg.Done()
    }()
    wg.Wait()

    //检查下单结果，回补交易差额，如果最后成交为0，开始下一轮
    if sorder.DealAmount > lorder.DealAmount {
        delta := sorder.DealAmount * sorder.AvgPrice - lorder.DealAmount * lorder.AvgPrice
        id := short.Buy(delta, 0)
        order := hg.orderInfo(id, short)
        sorder.DealAmount -= order.DealAmount
    } else {
        delta := lorder.DealAmount - sorder.DealAmount
        if id := long.Sell(0, delta); id > 0 {
            lorder.DealAmount -= delta
        }
    }

    short.amountChange -= sorder.DealAmount
    long.amountChange += lorder.DealAmount

    if sorder.DealAmount == 0 || lorder.DealAmount == 0 {
        hg.state = StateClose
    } else {
        hg.state = StateOpen
    }

    //交易统计
    gmvc.Logger.Println(fmt.Sprintf("   short: %v - %.4f btc + %.2f(%.2f) cny",
        short.name, sorder.DealAmount, shortSellPrice, sorder.AvgPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v + %.4f btc - %.2f(%.2f) cny",
        long.name, lorder.DealAmount, longBuyPrice, lorder.AvgPrice))
    gmvc.Logger.Println("")
}

func (hg *Hedge) openShort(short *Market, sellPrice float64) Order {
    hg.short = short
    var id int64

    //下单，如果失败重试2次
    for i := 0; i < 3; i++ {
        id = short.Sell(sellPrice, hg.tradeAmount)
        if id > 0 {
            break
        }
    }

    return hg.orderInfo(id, short)
}

func (hg *Hedge) openLong(long *Market, buyPrice float64) Order {
    hg.long = long
    var id int64

    //下单，如果失败重试2次
    for i := 0; i < 3; i++ {
        id = long.Buy(buyPrice, hg.tradeAmount)
        if id > 0 {
            break
        }
    }

    return hg.orderInfo(id, long)
}

/*
获取订单结果
 */
func (hg *Hedge) orderInfo(id int64, market *Market) Order {
    var order Order
    if id > 0 {
        //每隔0.5s读取一次，最多等带5s
        for i := 0; i < 10; i++ {
            time.Sleep(500 * time.Millisecond)
            order = market.OrderInfo(id)
            if order.Status == 2 {
                break
            }
        }

        //如果订单没有完全成交
        if order.Status != 2 {
            canceled := false

            //重试两次，如果都失败中断程序
            for i := 0; i < 2; i++ {
                canceled = market.CancelOrder(id)
            }

            if !canceled {
                gmvc.Logger.Println(fmt.Sprintf("cancel order failed v% order id = v%", market.Name(), id))
            }

            //更新order info
            for i := 0; i < 3; i++ {
                order = market.OrderInfo(id)
                if order.Id > 0 {
                    break
                }
            }

            if order.Id == 0 {
                gmvc.Logger.Println(fmt.Sprintf("update order info failed v% order id = v%", market.Name(), id))
            }
        }
    }

    return order
}


func (hg *Hedge) closePosition(buyPrice, sellPrice float64) {
    wg := *sync.WaitGroup{}
    wg.Add(2)

    var sorder, lorder Order
    go func() {
        sorder = hg.closeShort(buyPrice)
        wg.Done()
    }()


    go func() {
        lorder = hg.closeLong(sellPrice)
        wg.Done()
    }()
    wg.Wait()

    hg.state = StateClose
    hg.roundNum++

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
                                    hg.long.cnyChange + hg.short.cnyChange,
                                    hg.roundNum, d.Format("15:04:05")))
    gmvc.Logger.Println("")
}

func (hg *Hedge) closeShort(price float64) Order {
    id := hg.short.Buy(price, -hg.short.amountChange)
    return hg.orderInfo(id, hg.short)
}

func (hg *Hedge) closeLong(price float64) Order {
    id := hg.long.Sell(price, hg.long.amountChange)
    return hg.orderInfo(id, hg.long)
}

