package market

import (
    "github.com/roydong/gmvc"
    "container/list"
    "time"
)

var maxTickerNum = 200

type exchanger interface {
    Sell(amount float64) error

    Buy(price float64) error

    LastTicker() *Ticker

    GetBalance() (float64, float64)
}

type Ticker struct {
    High float64
    Low  float64
    Sell float64
    Buy  float64
    Last float64
    Vol  float64
    Time int64
}

type Market struct {
    exchanger

    name string
    tickers map[int64]*Ticker
    tickerList *list.List
    maxTickerNum int

    recentTotalPrince float64
    recentAverage float64

    btc float64
    cny float64
}


func NewMarket(name string) *Market {
    m := &Market{
        name: name,
        maxTickerNum: maxTickerNum,
        recentTotalPrince: 0,
        recentAverage: 0,
    }

    switch m.name {
    case "okcoin":
        m.exchanger = newOKCoin()
    case "huobi":
        m.exchanger = newHuobi()
    default:
        gmvc.Logger.Fatalln("invalid market " + m.name)
    }

    m.tickers = make(map[int64]*Ticker)
    m.tickerList = list.New()

    return m
}

func (m *Market) addTicker(t *Ticker) {
    m.tickerList.PushFront(t)
    m.tickers[t.Time] = t
    m.recentTotalPrince += t.Last
    if m.tickerList.Len() > m.maxTickerNum {
        el := m.tickerList.Back()
        m.tickerList.Remove(el)
        back, _ := el.Value.(Ticker)
        m.recentTotalPrince -= back.Last
        delete(m.tickers, back.Time)
    }

    m.recentAverage = m.recentTotalPrince / float64(m.tickerList.Len())

    //gmvc.Logger.Println(fmt.Sprintf("%v: %v, %.2f, %.2f, %v", m.name, m.tickerList.Len(), m.recentAverage, t.Last, t.Time))
}

func (m *Market) TickerByTime(t int64) *Ticker {
    return m.tickers[t]
}

func (m *Market) BackTicker() *Ticker {
    if el := m.tickerList.Back(); el != nil {
        return el.Value.(*Ticker)
    }
    return nil
}

func (m *Market) FrontTicker() *Ticker {
    if el := m.tickerList.Front(); el != nil {
        return el.Value.(*Ticker)
    }
    return nil
}

func (m *Market) SyncTicker(interval time.Duration) {
    for _ = range time.Tick(interval) {
        ticker := m.LastTicker()
        m.addTicker(ticker)
    }
}

func (m *Market) SyncBalance() {
    m.btc, m.cny = m.GetBalance()
}

func (m *Market) Balance() (float64, float64) {
    return m.btc, m.cny
}

