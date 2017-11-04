package mctree

import (
    "dw/poker/market/context"
    "math"
    "time"
    "dw/poker/market"
    "fmt"
    "strings"
)

type State struct {
    startTime time.Time
    basePrice float64
    kPrices []int
    price int
    profit int
}

func newState(klines []context.Kline, price float64, profit int) *State {
    s := &State{}
    s.startTime = klines[0].OpenTime
    s.basePrice = klines[0].AvgPrice
    s.kPrices = make([]int, 0, len(klines))
    for _, kline := range klines {
        s.kPrices = append(s.kPrices, s.getPriceRate(kline.AvgPrice, 1000))
    }
    s.price = s.getPriceRate(price, 1000)
    s.profit = profit
    return s
}

func (s *State) isSame(s1 *State) bool {
    return s.key() == s1.key()
}

func (s *State) getPriceRate(price float64, scale float64) int {
    return int(math.Floor((price - s.basePrice) / s.basePrice * scale))
}

func (s *State) key() string {
    vals := make([]string, 0, len(s.kPrices))
    for _, p := range s.kPrices {
        vals = append(vals, fmt.Sprintf("%d", p))
    }
    return fmt.Sprintf("%s-%d-%d", strings.Join(vals, "-"), s.price, s.profit)
}

type Action struct {
    Cmd context.TradeAction
    Amount int
}

var ex *market.Exchange

var forest map[string]*MCTree

type mcChild struct {
    action Action
    mctree *MCTree
}

type MCTree struct {
    id int
    score int
    state State
    parent *MCTree
    children []mcChild
}

func (m *MCTree) transform(a Action) *MCTree {
    //action  make order

    //check till the state changed
    //if has order must wait order completed or canceled

    //return new state
}

func (m *MCTree) backPropagate(r int) {
    if m.parent != nil {
        m.parent.score += r
        m.parent.backPropagate(r)
    }
}

func (m *MCTree) reward() int {
    return m.state.profit
}





