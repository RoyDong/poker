package worm

import (
    "dw/poker/market"
    "fmt"
    "strings"
    "sort"
    "log"
    "time"
    "dw/poker/utils"
    "dw/poker/market/context"
)


type worm struct {
    state []int
    transforms []*transform
}

func newWorm(prices []float64) *worm {
    if len(prices) == 0 {
        return nil
    }
    basePrice := prices[0]
    state := make([]int, 0, len(prices))
    for _, p := range prices {
        r := int((p - basePrice) / basePrice * 1000)
        if r > 1000 {
            r = 1000
        } else if r < -1000 {
            r = -1000
        }
        state = append(state, r)
    }
    w := &worm{}
    w.state = state
    if wm, has := forest[w.key()]; has {
        w = wm
    } else {
        forest[w.key()] = w
    }
    return w
}

func (w *worm) key() string {
    vals := make([]string, 0, len(w.state))
    for _, p := range w.state {
        vals = append(vals, fmt.Sprintf("%d", p))
    }
    return strings.Join(vals, ">")
}

func (w *worm) transform(prices []float64) *worm {
    nw := newWorm(prices)
    for _, trans := range w.transforms {
        if trans.result.key() == nw.key() {
            trans.freq += 1
            sort.Sort(transSort(w.transforms))
            return nw
        }
    }
    w.transforms = append(w.transforms, &transform{freq: 1, result: nw})
    return nw
}

type transform struct {
    freq int
    result *worm
}

type transSort []*transform

func (s transSort) Len() int {
    return len(s)
}

func (s transSort) Less(i, j int) bool {
    return s[i].freq > s[j].freq
}

func (s transSort) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}

var forest = make(map[string]*worm, 0)

var WormSize = 30

func Start() {
    okex := market.GetExchange(market.OkexQuarter)

    loadFromDb()

    prices := make([]float64, 0, WormSize)
    var worm *worm
    okex.AddHandler("kline_close", func(args ...interface{}) {
        kline, _ := args[0].(*context.Kline)

        prices = append(prices, kline.AvgPrice)
        if len(prices) == 30 {
            if worm == nil {
                worm = newWorm(prices)
            } else {
                worm = worm.transform(prices)
            }
        }
    })
}


func loadFromDb() {
    sql := "select avg_price from kline where exname = ?"
    r, err := utils.MainDB.Query(sql, market.OkexQuarter)
    if err != nil {
        utils.FatalLog.Write("load prices from kline %s", err.Error())
    }

    var allPrices []float64
    utils.Scan(r, &allPrices)

    prices := make([]float64, 0, WormSize)
    var worm *worm
    for i := 0; i < len(allPrices); i++ {
        if allPrices[i] > 0 {
            prices = append(prices, allPrices[i])
        }
        if len(prices) == 30 {
            if worm == nil {
                worm = newWorm(prices)
            } else {
                worm = worm.transform(prices)
            }
        }
    }
}







