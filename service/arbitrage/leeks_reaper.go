package arbitrage

import (
    "time"
    "github.com/roydong/gmvc"
)

type LeeksReaper struct {
    exchange *Exchange

    balance Balance

    state int

    bidPrice, askPrice float64

    prices []float64
    trades []Trade
    vol float64
    initPosition bool
}

func NewLeeksReaper(ex *Exchange) *LeeksReaper {
    lr := &LeeksReaper{
        exchange: ex,
        initPosition: true,
        state: StateStop,
    }
    return lr
}

func (lr *LeeksReaper) Start() {
    lr.state = StateOpen
    for lr.state == StateOpen {
        lr.balancePosition()
    }
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
根据盘口预估交易价格
 */
func (lr *LeeksReaper) updatePrice() {
    asks, bids := lr.exchange.GetDepth()
    lr.bidPrice = bids[0][0] * 0.618 + asks[0][0] * 0.382 + 0.01
    lr.askPrice = bids[0][0] * 0.382 + asks[0][0] * 0.618 - 0.01
    price := (bids[0][0] + asks[0][0]) * 0.35 +
             (bids[1][0] + asks[1][0]) * 0.10 +
             (bids[2][0] + asks[2][0]) * 0.05
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
    lr.updatePrice()
    //开始计算仓位
    price := lr.prices[len(lr.prices) - 1]
    shortValue := lr.balance.ShortAmount * 100 //空头仓位价值
    totalValue := lr.balance.AccountRights * price //多头仓位价值
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
            order = lr.exchange.Trade(OpenShortPosition, 1, lr.askPrice)
        } else if position > 0.52 {
            order = lr.exchange.Trade(CloseShortPosition, 1, lr.bidPrice)
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
func (lr *LeeksReaper) follow() {

}


