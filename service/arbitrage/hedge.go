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

        margin: newAverager(600),

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
        idx := now.Second()
        wg.Add(2)
        go func() {
            hg.zuo.calcMgm()
            wg.Done()
        }()
        go func() {
            hg.you.calcMgm()
            wg.Done()
        }()
        wg.Wait()

        hg.margin.Add(idx, hg.you.mgm - hg.zuo.mgm)
        log.Println(fmt.Sprintf("%.2f <= %.2f => %.2f", hg.margin.Min(), hg.margin.Avg(), hg.margin.Max()))
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
            //log.Println(fmt.Sprintf("margin: sell %.2f max %.2f", margin, hg.highMargin()))

            //满足最小差价限制,并且超过最大差价
            if margin >= hg.margin.Avg() + hg.minTradeMargin && margin >= hg.margin.Max() {
                gmvc.Logger.Println(fmt.Sprintf("open positoin(youSell - zuoBuy %.2f):", margin ))
                hg.tradeAmount = math.Min(math.Min(youSellAmount, zuoBuyAmount), hg.maxTradeAmount)
                hg.openPosition(hg.you, youSellPrice, hg.zuo, zuoBuyPrice)
                continue
            }

            //尝试判断是否可以左手做空(右手多), 以右手的最近卖单价 - 左手的买单价(margin)和(min max avg)相关参数比较
            margin = youBuyPrice - zuoSellPrice
            //log.Println(fmt.Sprintf("margin: buy %.2f min %.2f", margin, hg.lowMargin()))

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
        lorder = hg.openLong(long, longBuyPrice)
        wg.Done()
    }()
    go func() {
        sorder = hg.openShort(short, shortSellPrice)
        wg.Done()
    }()
    wg.Wait()

    /*
    检查下单结果，补齐对冲敞口，暂时使用市价交易
    1. 当空头交易过多，则空方市价平仓买入差额
    2. 当多头交易过多，则多方市价平仓卖出差额
     */
    if sorder.DealAmount > lorder.DealAmount + 0.001 {
        delta := sorder.DealAmount * sorder.AvgPrice - lorder.DealAmount * lorder.AvgPrice
        if id := short.CloseShort(delta, 0); id > 0 {
            order := hg.closeOrder(id, short)
            sorder.DealAmount -= order.DealAmount
        }
    } else if lorder.DealAmount > sorder.DealAmount + 0.001 {
        delta := lorder.DealAmount - sorder.DealAmount
        if id := long.CloseLong(0, delta); id > 0 {
            lorder.DealAmount -= delta
        }
    }

    short.amountChange -= sorder.DealAmount
    long.amountChange += lorder.DealAmount

    //交易统计
    gmvc.Logger.Println(fmt.Sprintf("   short: %v - %.4f btc + %.2f(%.2f) cny",
        short.name, sorder.DealAmount, shortSellPrice, sorder.AvgPrice))
    gmvc.Logger.Println(fmt.Sprintf("   long: %v + %.4f btc - %.2f(%.2f) cny",
        long.name, lorder.DealAmount, longBuyPrice, lorder.AvgPrice))

    if sorder.DealAmount <= 0.001 || lorder.DealAmount <= 0.001 {
        gmvc.Logger.Println("closed")
    } else {
        hg.state = StateOpen
        gmvc.Logger.Println("openned")
    }
}

func (hg *Hedge) openShort(short *Exchange, sellPrice float64) Order {
    hg.short = short
    var id int64

    //下单，如果失败重试2次
    for i := 0; i < 3; i++ {
        id = short.OpenShort(sellPrice, hg.tradeAmount + short.amountChange) //加上上次交易后的差额
        if id > 0 {
            short.amountChange = 0
            break
        }
    }

    return hg.closeOrder(id, short)
}

func (hg *Hedge) openLong(long *Exchange, buyPrice float64) Order {
    hg.long = long
    var id int64

    //下单，如果失败重试2次
    for i := 0; i < 3; i++ {
        id = long.OpenLong(buyPrice, hg.tradeAmount)
        if id > 0 {
            break
        }
    }

    return hg.closeOrder(id, long)
}

/*
完结订单，获取订单结果
 */
func (hg *Hedge) closeOrder(id int64, market *Exchange) Order {
    var order Order
    if id > 0 {
        //每隔0.5s读取一次，最多等带5s
        for i := 0; i < 10; i++ {
            time.Sleep(500 * time.Millisecond)
            order = market.Order(id)
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
                gmvc.Logger.Println(fmt.Sprintf("cancel order failed %v order id = %v", market.Name(), id))
            }

            //更新order info
            for i := 0; i < 3; i++ {
                order = market.Order(id)
                if order.Id > 0 {
                    break
                }
            }

            if order.Id == 0 {
                gmvc.Logger.Println(fmt.Sprintf("update order info failed %v order id = %v", market.Name(), id))
            }
        }
    }

    return order
}


func (hg *Hedge) closePosition(buyPrice, sellPrice float64) {
    if hg.test {
        return
    }

    wg := &sync.WaitGroup{}
    wg.Add(2)

    var sorder, lorder Order
    go func() {
        hg.closeShort(buyPrice)
        wg.Done()
    }()
    go func() {
        hg.closeLong(sellPrice)
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

func (hg *Hedge) closeShort(price float64) {
    amount := -hg.short.amountChange
    id := hg.short.CloseShort(price, amount)
    order := hg.closeOrder(id, hg.short)

    //未完全成交差额使用市价交易来回补
    if order.DealAmount < amount {
        delta := (amount - order.DealAmount) * order.AvgPrice
        if id := hg.short.CloseShort(delta, 0); id > 0 {
            o := hg.closeOrder(id, hg.short)
            order.DealAmount += o.DealAmount
        }
    }
    hg.short.amountChange += order.DealAmount
}

func (hg *Hedge) closeLong(price float64) {
    amount := hg.long.amountChange
    id := hg.long.CloseLong(price, amount)
    order := hg.closeOrder(id, hg.long)

    //未完全成交差额使用市价交易来回补
    if order.DealAmount < amount {
        delta := amount - order.DealAmount
        if id := hg.long.CloseLong(0, delta); id > 0 {
            order.DealAmount -= amount
        }
    }
    hg.short.amountChange = 0
}

