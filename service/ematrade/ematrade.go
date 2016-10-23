package ematrade

import (
    "time"
    "github.com/roydong/poker/service/market"
    "container/list"
    "math"
)


const (
    StateStop = 1
    StateRunning = 2
)

/*
期望平均值交易,假定交易都会在均线附近震荡,利用价格的波动获利
 */
type EMATrade struct {
    market *market.Market

    long, short *position

    //多头开仓次数,空头开仓次数
    longNum, shortNum int

    profit float64

    startTime, endTime time.Time

    //保存最新的成交价格
    tradePrices *list.List

    //ema值
    emaPoints *list.List

    emaLength int

    //检查时间间隔
    interval float64

    priceInterval float64

    //当前的ema斜率
    currentSlope float64

    //交易价格和ema值的差值
    minMargin, maxMargin float64
    minMarginTime, maxMarginTime time.Time

    tradeAmount float64

    state int
}



func (et *EMATrade) Run() {

}


/*
获取最近n次交易的几何平均数
 */
func (et *EMATrade) getGeomean() float64 {
    n := 0
    product := 1
    for el := et.tradePrices.Front(); el != nil; el = el.Next() {
        n++
        product *= el.Value.(int64)
    }
    return math.Pow(product, 1 / n)
}


/*
差价检查,如果满足条件则开始操作
 */
func (et *EMATrade) checkSpread() {

}

/*
计算当前ema线的顺时斜率
 */
func (et *EMATrade) getSlope() {

}

/*
计算出当前的ema值, 取最近n个交易价格进行计算
 */
func (et *EMATrade) getCurrentEMA(n int) {

}

func (et *EMATrade) checkLoop() {
    for et.state == StateRunning {
        time.Sleep(et.interval * time.Second)
        price := et.market.LastTicker().Last
        if price <= 0 {
            continue
        }

        //获取最新交易价格
        et.tradePrices.PushFront(price)
        if et.tradePrices.Len() > et.emaLength {
            el := et.tradePrices.Back()
            et.tradePrices.Remove(el)
        }

        //根据最新的n次价格计算初当前的几何平均值
        mean := et.getGeomean()
        et.emaPoints.PushFront(mean)
        if et.emaPoints.Len() > et.emaLength {
            el := et.emaPoints.Back()
            et.emaPoints.Remove(el)
        }

        //计算均线斜率
        if et.emaPoints.Len() > 2 {
            el := et.emaPoints.Front()
            mean1 := el.Value.(float64)
            mean2 := el.Next().Value.(float64)
            if mean1 != mean2 {
                et.currentSlope = mean1 - mean2 / et.interval
            }
        }
    }
}

func (et *EMATrade) tradeLoop() {
    for et.state == StateRunning && et.emaPoints.Len() >= et.emaLength {
        time.Sleep(et.interval * time.Second)

        price := et.tradePrices.Front().Value.(float64)
        mean := et.emaPoints.Front().Value.(float64)
        margin := price - mean

        if margin < et.minMargin {
            et.minMargin = margin
            et.minMarginTime = time.Now()
        } else if margin > et.maxMargin {
            et.maxMargin = margin
            et.maxMarginTime = time.Now()
        }

        if et.minMarginTime.Unix() + 60 < time.Now().Unix() && et.minMargin < -et.priceInterval {
            et.minMargin += et.priceInterval
        }
        if et.maxMarginTime.Unix() + 60 < time.Now().Unix() && et.maxMargin > et.priceInterval {
            et.maxMargin -= et.priceInterval
        }

        if margin < -et.priceInterval && math.Abs(margin - et.minMargin) <= et.priceInterval {

        } else if margin > et.priceInterval && math.Abs(margin - et.maxMargin) <= et.priceInterval {

        }
    }
}


/*
开多单买入，买入数量要减去之前没有平仓的数量
等待成交完成或者价格恢复到均价
取消订单
在均线附近卖出，等待完全成交，或者价格偏离
*/
func (et *EMATrade) openLong(price float64) {
    id := et.market.Buy(et.tradeAmount, price)
    //交易统计
    var order market.Order
    for _ = range time.Tick(500 * time.Millisecond) {
        order = et.market.OrderInfo(id)
        currentPrice := et.tradePrices.Front().Value.(float64)
        if order.Status == 2 || order.Status == -1 {
            et.long = &position{
                amount: order.DealAmount,
                price: order.AvgPrice,
                postionType: TypeLong,
                state: StateOpen,
                openAt: time.Now(),
            }
            break
        } else if currentPrice > price {
            et.market.CancelOrder(id)
        }
    }
}

func (et *EMATrade) openShort() {

}



