package worm

import (
    "dw/poker/market"
    "fmt"
    "strings"
    "sort"
    "log"
    "time"
)

var ex *market.Exchange

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

func (w *worm) transform(prices []float64) {
    nw := newWorm(prices)
    for _, trans := range w.transforms {
        if trans.result.key() == nw.key() {
            trans.freq += 1
            sort.Sort(transSort(w.transforms))
            return
        }
    }
    w.transforms = append(w.transforms, &transform{freq: 1, result: nw})
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


func Start() {
    time.Sleep(10 * time.Second)
    ok := market.GetExchange("okex/quarter")
    trades := ok.LastTrades()
    prices := make([]float64, 0, len(trades))
    for _, t := range trades {
        if t.Price > 0 {
            prices = append(prices, t.Price)
        }
    }
    worm := newWorm(prices[0:10])
    for i := 1; i < len(prices) - 10; i++ {
        worm.transform(prices[i:i+10])
    }
    for k, v := range forest {
        log.Println("============")
        log.Println(k)
        for _, t := range v.transforms {
            log.Println("----", t.freq, t.result.key())
        }
    }
}



