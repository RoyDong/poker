package market

import (
    "time"
)



type position struct {
    amount float64

    openPrice float64

    closePrice float64

    closeInfo []float64

    //持仓类型  long/short
    typ string

    created time.Time
}


/*
平均值交易,暂定交易都会在均线附近震荡
 */
type MeanTrade struct {
    market *Market

    //多头,空头持仓量
    longAmount, shortAmount float64

    longProfit, shortProfit, totalProfit float64

    //多头开仓次数,空头开仓次数
    longNum, shortNum int

    startTime, endTime time.Time
}





func (mt *MeanTrade) Run() {

}


/*
获取最近n次交易的几何平均数
 */
func (mt *MeanTrade) getGeomean(n int) {

}



/*
差价检查,如果满足条件则开始操作
 */
func (mt *MeanTrade) checkSpread() {

}


