package arbitrage

import (
    "time"
    "github.com/roydong/gmvc"
    "fmt"
    "math"
)

type LeeksReaper struct {
    exchange *Exchange

    balance Balance

    state int

    bidPrice, askPrice float64

    asks, bids [][]float64
    prices []float64
    trades []Trade
    vol float64
    initPosition bool

    minTradeAmount float64

    burstThresholdPct, burstThresholdVol float64
}

func NewLeeksReaper(ex *Exchange) *LeeksReaper {
    lr := &LeeksReaper{
        exchange: ex,
        initPosition: true,
        state: StateStop,

        minTradeAmount: 1,
    }
    return lr
}

/*
更新最近的交易，计算出当前的交易量
 */
func (lr *LeeksReaper) updateTrades() {
    lr.trades = lr.exchange.GetTrades()
    length := len(lr.trades)
    lr.prices = make([]float64, 0, length + 1)
    for i := 0; i < len(lr.trades); i++ {
        lr.prices = append(lr.prices, lr.trades[i].Price)
    }
    lr.vol = lr.vol * 0.7 + lr.trades[length - 1].Amount
}

/*
更新盘口数据
 */
func (lr *LeeksReaper) updateOrderBook() {
    lr.asks, lr.bids = lr.exchange.GetDepth()
    lr.bidPrice = lr.bids[0][0] * 0.618 + lr.asks[0][0] * 0.382 + 0.01
    lr.askPrice = lr.bids[0][0] * 0.382 + lr.asks[0][0] * 0.618 - 0.01
    price := (lr.bids[0][0] + lr.asks[0][0]) * 0.35 +
             (lr.bids[1][0] + lr.asks[1][0]) * 0.10 +
             (lr.bids[2][0] + lr.asks[2][0]) * 0.05
    lr.prices = append(lr.prices, price)
}

/*
平衡仓位保持对冲平衡
 */
func (lr *LeeksReaper) balancePosition() {
    if lr.balance = lr.exchange.GetBalance(); lr.balance.AccountRights == 0 {
        return
    }
    lr.updateTrades()
    lr.updateOrderBook()
    //开始计算仓位
    price := lr.prices[len(lr.prices) - 1]
    shortValue := lr.balance.ShortAmount * 100 //空头仓位价值
    totalValue := lr.balance.AccountRights * price //保证金价值
    var order Order
    oids := make([]int64, 0, 5)
    //第一次需要初始化仓位，保持50%
    if lr.initPosition {
        //需要增加(负数表示减少)的空头仓位
        amount := (totalValue - shortValue) / 100
        gmvc.Logger.Println(amount, price, shortValue, totalValue, 1111)
        if amount > 0 {
            order = lr.exchange.Trade(OpenShortPosition, amount, lr.askPrice)
        } else if amount < 0 {
            order = lr.exchange.Trade(CloseShortPosition, amount, lr.bidPrice)
        }
        if order.Id > 0 {
            oids = append(oids, order.Id)
        }
        if order.DealAmount >= amount {
            lr.initPosition = false
        }
    } else {
        position := shortValue / totalValue
        if position < 0.48 {
            order = lr.exchange.Trade(OpenShortPosition, lr.minTradeAmount, lr.askPrice)
        } else if position > 0.52 {
            order = lr.exchange.Trade(CloseShortPosition, lr.minTradeAmount, lr.bidPrice)
        }
        if order.Id > 0 {
            oids = append(oids, order.Id)
        }
    }
    time.Sleep(1 * time.Second)
    lr.cancelOrders(oids)
}

func (lr *LeeksReaper) cancelOrders(ids []int64) {
    for _, id := range ids {
        lr.exchange.CancelOrder(id)
    }
}

/*
追随趋势下单
 */
func (lr *LeeksReaper) Start() {
    lr.state = StateOpen
    numTick := 0
    for lr.state == StateOpen {
        lr.updateTrades()
        lr.updateOrderBook()
        lr.balancePosition()

        var bull, bear bool
        priceLen := len(lr.prices)
        lastPrice := lr.prices[priceLen - 1]
        burstPrice := lastPrice * lr.burstThresholdPct
        if lr.balance.AccountRights > 0 {
            gmvc.Logger.Println(fmt.Sprintf("Tick: %v, lastPrice: %v, burstPrice %.2f", numTick, lastPrice, burstPrice))
        }

        if numTick > 2 {
            if lastPrice - max(lr.prices[priceLen - 6:priceLen - 2]...) > burstPrice ||
            lastPrice - max(lr.prices[priceLen - 6:priceLen - 3]...) > burstPrice &&
            lastPrice > lr.prices[priceLen - 2] {
                bull = true
            } else if lastPrice - min(lr.prices[priceLen - 6:priceLen - 2]...) < -burstPrice ||
            lastPrice - min(lr.prices[priceLen - 6:priceLen - 3]...) < - burstPrice &&
            lastPrice < lr.prices[priceLen - 2] {
                bear = true
            }
        }

        amount := lr.balance.ShortAmount
        //成交量小，减少力度
        if lr.vol < lr.burstThresholdVol {
            amount = amount * lr.vol / lr.burstThresholdVol
        }
        //次数
        if numTick < 5 {
            amount *= 0.8 * 0.8
        } else if amount < 10 {
            amount *= 0.8
        }
        //当前价格与突破方向不明显，涨时当前价格不是最近的最高价，跌时当前价格不是最近的最底价
        if bull && lastPrice < max(lr.prices...) {
            amount *= 0.9
        }
        if bear && lastPrice > min(lr.prices...) {
            amount *= 0.9
        }
        //最近2次价格变动较大
        if math.Abs(lastPrice - lr.prices[priceLen - 2]) > burstPrice * 2 {
            amount *= 0.9
        }
        if math.Abs(lastPrice - lr.prices[priceLen - 2]) > burstPrice * 3 {
            amount *= 0.9
        }
        if math.Abs(lastPrice - lr.prices[priceLen - 2]) > burstPrice * 4 {
            amount *= 0.9
        }
        //盘口差价较大
        if lr.asks[0][0] - lr.bids[0][0] > burstPrice * 2 {
            amount *= 0.9
        }
        if lr.asks[0][0] - lr.bids[0][0] > burstPrice * 3 {
            amount *= 0.9
        }
        if lr.asks[0][0] - lr.bids[0][0] > burstPrice * 4 {
            amount *= 0.9
        }

        for amount >= 1 {
            var order Order
            if bull {
                order = lr.exchange.Trade(CloseShortPosition, amount, lr.bidPrice)
            } else {
                order = lr.exchange.trades(OpenShortPosition, amount, lr.askPrice)
            }
            if order.Id > 0 {
                time.Sleep(200 * time.Millisecond)
                lr.exchange.CancelOrder(order.Id)
            }
            amount -= order.DealAmount
            amount *= 0.98
        }
        numTick++
    }
}

func (lr *LeeksReaper) Stop() {
    lr.state = StateStop
}


