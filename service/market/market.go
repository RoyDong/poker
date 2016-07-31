package market

import (
    "github.com/roydong/gmvc"
    "container/list"
    "time"
)

var maxTickerNum = 300

type exchanger interface {
    Sell(amount float64) int64

    Buy(price float64) int64

    OrderInfo(id int64) Order

    LastTicker() Ticker

    GetDepth() ([][]float64, [][]float64)

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

type Order struct {
    Id int64
    Type int
    Amount float64
    Price float64
    Created int64

    DealAmount float64
    AvgPrice float64
    Status int
    Fee float64
}

type Market struct {
    exchanger

    name string
    tickers map[int64]Ticker
    tickerList *list.List
    maxTickerNum int

    recentTotalPrince float64
    recentAverage float64

    amount float64
    amountChange float64
    cny float64
    cnyChange float64

    lastAsks [][]float64
    lastBids [][]float64
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
        m.exchanger = NewOKCoin()
    case "huobi":
        m.exchanger = NewHuobi()
    case "haobtc":
        m.exchanger = NewHaobtc()

    default:
        gmvc.Logger.Fatalln("invalid market " + m.name)
    }

    m.tickers = make(map[int64]Ticker)
    m.tickerList = list.New()

    return m
}

func (m *Market) Name() string {
    return m.name
}

func (m *Market) addTicker(t Ticker) {
    if t.Last <= 0 {
        return
    }
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
}

func (m *Market) TickerByTime(t int64) Ticker {
    return m.tickers[t]
}

func (m *Market) BackTicker() Ticker {
    if el := m.tickerList.Back(); el != nil {
        return el.Value.(Ticker)
    }
    return Ticker{}
}

func (m *Market) FrontTicker() Ticker {
    if el := m.tickerList.Front(); el != nil {
        return el.Value.(Ticker)
    }
    return Ticker{}
}

func (m *Market) SyncTicker(interval time.Duration) {
    for _ = range time.Tick(interval) {
        ticker := m.LastTicker()
        m.addTicker(ticker)
    }
}

/*
根据深度获得需要卖出amount数量的btc需要的价位,即获取对应的买单价(bid price)
 */
func (m *Market) GetSellPrice(amount float64) float64 {
    var sum, price float64
    for _, bid := range m.lastBids {
        price = bid[0]
        sum += bid[1]
        if sum >= amount {
            break
        }
    }
    return price
}

/*
根据深度获得需要买入amount数量的btc需要的价位,即获取对应的卖单价(ask price)
 */
func (m *Market) GetBuyPrice(amount float64) float64 {
    var sum, price float64
    for _, ask := range m.lastAsks {
        price = ask[0]
        sum += ask[1]
        if sum >= amount {
            break
        }
    }
    return price
}

func (m *Market) UpdateDepth() {
    lastAsks, lastBids := m.GetDepth()
    if len(lastAsks) > 0 && len(lastBids) > 0 {
        m.lastAsks = lastAsks
        m.lastBids = lastBids
    }
}

func (m *Market) SyncBalance() {
    m.amount, m.cny = m.GetBalance()
}

func (m *Market) Balance() (float64, float64) {
    return m.amount, m.cny
}

