package ematrade

import (
    "time"
    "github.com/roydong/poker/service/market"
    "container/list"
)



/*
期望平均值交易,暂定交易都会在均线附近震荡,利用价格的波动获利
 */
type EMATrade struct {
    market *market.Market

    long, short *position

    //多头开仓次数,空头开仓次数
    longNum, shortNum int

    profit float64

    startTime, endTime time.Time

    //按照一个较短的时间间隔保存ema值
    emaPoints *list.List

    //保存最新的成交价格
    tradePrices *list.List

    //交易价格和ema值的差值
    minMargin, maxMargin float64
}




func (et *EMATrade) Run() {

}


/*
获取最近n次交易的几何平均数
 */
func (et *EMATrade) getGeomean(n int) {

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
