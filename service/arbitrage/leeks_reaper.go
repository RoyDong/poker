package arbitrage

import (
    "time"
    "fmt"
    "math"
    "log"
)

type LeeksReaper struct {
    exchange *Exchange

    balance Balance

    state int

    bidPrice, askPrice float64

    //最新盘口深度
    asks, bids [][]float64
    //最近几次成交价格
    prices []float64
    //最近几次成交记录
    trades []Trade
    //最近一次交易量
    vol float64
    //最大总持仓量
    maxPositionAmount float64
    //一次最小交易量
    minTradeAmount float64
    //突破价格百分比阈值, 突破交易量阈值
    burstThresholdPct, burstThresholdVol float64
}

func NewLeeksReaper(ex *Exchange) *LeeksReaper {
    lr := &LeeksReaper{
        exchange: ex,
        state: StateStop,
        minTradeAmount: 1,
        maxPositionAmount: 100,
        burstThresholdPct: 0.0005,
        burstThresholdVol: 100,
    }
    return lr
}

/*
更新最近的交易，计算出当前的交易量
 */
func (lr *LeeksReaper) updateTrades() {
    trades := lr.exchange.GetTrades()
    length := len(trades)
    if length > 0 {
        lr.trades = trades
        lr.prices = make([]float64, 0, length + 1)
        for i := 0; i < length; i++ {
            lr.prices = append(lr.prices, lr.trades[i].Price)
        }
        lr.vol = lr.vol * 0.7 + lr.trades[length - 1].Amount
    }
}

/*
更新盘口数据
 */
func (lr *LeeksReaper) updateOrderBook() {
    asks, bids := lr.exchange.GetDepth()
    if len(asks) > 0 && len(bids) > 0 {
        lr.asks = asks
        lr.bids = bids
        lr.bidPrice = lr.bids[0][0] * 0.618 + lr.asks[0][0] * 0.382 + 0.01
        lr.askPrice = lr.bids[0][0] * 0.382 + lr.asks[0][0] * 0.618 - 0.01
        price := (lr.bids[0][0] + lr.asks[0][0]) * 0.35 +
        (lr.bids[1][0] + lr.asks[1][0]) * 0.10 +
        (lr.bids[2][0] + lr.asks[2][0]) * 0.05
        lr.prices = append(lr.prices, price)
    }
}

/*
更新账户余额
 */
func (lr *LeeksReaper) updateBalance() {
    b := lr.exchange.GetBalance()
    if b.AccountRights > 0 {
        lr.balance = b
    }
}

/*
平衡仓位保持对冲平衡
 */
func (lr *LeeksReaper) balancePosition() {
    for lr.state == StateOpen {
        time.Sleep(1 * time.Second)
        lr.updateBalance()
        lr.updateTrades()
        lr.updateOrderBook()
        log.Println(fmt.Sprintf("Position: %.0f(%.4f)/%.0f(%.4f), Total: %.4f",
                lr.balance.LongAmount, lr.balance.LongProfit,
                lr.balance.ShortAmount, lr.balance.ShortProfit, lr.balance.AccountRights))
        if lr.balance.LongAmount - lr.balance.ShortAmount >= lr.minTradeAmount {
            lr.exchange.Trade(CloseLongPosition, lr.minTradeAmount, lr.askPrice)
        } else if lr.balance.ShortAmount - lr.balance.LongAmount >= lr.minTradeAmount {
            lr.exchange.Trade(CloseShortPosition, lr.minTradeAmount, lr.bidPrice)
        }
    }
}

/*
追随趋势下单
 */
func (lr *LeeksReaper) followTrend() {
    numTick := 0
    for lr.state == StateOpen {
        lr.updateBalance()
        lr.updateTrades()
        lr.updateOrderBook()
        var bull, bear bool
        var arrow string
        var amount float64
        priceLen := len(lr.prices)
        lastPrice := lr.prices[priceLen - 1]
        burstPrice := lastPrice * lr.burstThresholdPct
        if numTick > 2 {
            if lastPrice - max(lr.prices[priceLen - 6:priceLen - 2]...) > burstPrice ||
                    lastPrice - max(lr.prices[priceLen - 6:priceLen - 3]...) > burstPrice &&
                    lastPrice > lr.prices[priceLen - 2] {
                amount = math.Min(lr.maxPositionAmount / 2, lr.maxPositionAmount * 0.8 - lr.balance.LongAmount)
                bull = true
                arrow = "↑"
            } else if lastPrice - min(lr.prices[priceLen - 6:priceLen - 2]...) < -burstPrice ||
                    lastPrice - min(lr.prices[priceLen - 6:priceLen - 3]...) < - burstPrice &&
                    lastPrice < lr.prices[priceLen - 2] {
                amount = math.Min(lr.maxPositionAmount / 2, lr.maxPositionAmount * 0.8 - lr.balance.ShortAmount)
                bear = true
                arrow = "↓"
            }
        }

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
        //开始下单
        if numTick > 2 && amount >= 1 {
            openAmount := math.Min(lr.maxPositionAmount - lr.balance.ShortAmount - lr.balance.LongAmount, amount)
            log.Println(fmt.Sprintf("Tick: %v, Price: %.2f, BurstPrice %.2f, Amount: %.0f/%.0f%v\n",
                                    numTick, lastPrice, burstPrice, openAmount, amount, arrow))
            for amount >= 1 {
                var order Order
                if bull {
                    if openAmount > 1 {
                        order = lr.exchange.Trade(OpenLongPosition, openAmount, lr.bidPrice)
                        openAmount -= order.DealAmount
                    } else if amount > 1 {
                        order = lr.exchange.Trade(CloseShortPosition, amount, lr.bidPrice)
                    }
                }
                if bear {
                    if openAmount > 1 {
                        order = lr.exchange.Trade(OpenShortPosition, openAmount, lr.askPrice)
                        openAmount -= order.DealAmount
                    } else if amount > 1 {
                        order = lr.exchange.Trade(CloseLongPosition, amount, lr.askPrice)
                    }
                }
                amount -= order.DealAmount
            }
        }
        numTick++
        time.Sleep(1 * time.Second)
    }
}

func (lr *LeeksReaper) Start() {
    lr.state = StateOpen
    go lr.balancePosition()
    go lr.followTrend()
}

func (lr *LeeksReaper) Stop() {
    lr.state = StateStop
}


