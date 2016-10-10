package ematrade

import (
    "time"
    "github.com/roydong/poker/service/market"
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


