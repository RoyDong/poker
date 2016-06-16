package marginhedge

import "time"

type Exchange interface {

    Sell()

    Buy()

    Price(t time.Time) float64

    Average() float64

}


func GetMargins(ex1, ex2 Exchange) {
    margins := make(map[int64]int64)
    for t := range time.Tick(10 * time.Second) {
        price1 := ex1.Price(t)
        price2 := ex2.Price(t)

        margins[t.Unix()] = price1 - price2
    }
}


func arbitrage(ex1, ex2 Exchange) {
    now := time.Now()
    p1 := ex1.Price(now)
    p2 := ex2.Price(now)

    if p1 < p2
}





