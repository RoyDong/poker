package context

import (
    "time"
    "fmt"
)

/*
货币单位
 */
type CurrencyUnit string

/*
仓位（头寸）方向
 */
type PositionType string

/*
交易动作
 */
type TradeAction string

const (
    Long  PositionType = "Long"
    Short PositionType = "Short"

    OpenLong   TradeAction = "OpenLong"
    OpenShort  TradeAction = "OpenShort"
    CloseLong  TradeAction = "CloseLong"
    CloseShort TradeAction = "CloseShort"

    Buy  = TradeAction("Buy")
    Sell = TradeAction("Sell")

    CNY = CurrencyUnit("CNY")
    USD = CurrencyUnit("USD")
    BTC = CurrencyUnit("BTC")
    LTC = CurrencyUnit("LTC")

    OrderStatusCreated   = 0
    OrderStatusPartial   = 1 //部分成交
    OrderStatusComplete  = 2
    OrderStatusCanceled  = 3
    OrderStatusCanceling = 4
)

type Order struct {
    Id      string
    TAction TradeAction
    Amount  float64
    Price   float64

    DealAmount float64
    DealMoney  float64
    AvgPrice   float64
    Fee        float64
    Status     int

    CreateTime time.Time
}

type Trade struct {
    Id         string
    TAction    TradeAction
    Amount     float64
    Price      float64
    Fee        float64
    CreateTime time.Time
}

type Position struct {
    Id              string
    PType           PositionType
    Amount          float64
    AvailableAmount float64
    AvgPrice        float64
    Money           float64
    Deposit         float64
    Leverage        float64
    ForceClosePrice float64
}

func (this Position) GetProfit(price float64) float64 {
    profit := this.Amount * (price - this.AvgPrice)
    if this.PType == Long {
        return profit
    }
    return -profit
}

func (this Position) GetROP(price float64) float64 {
    if this.Deposit > 0 {
        return this.GetProfit(price) / this.Deposit
    }
    return 0
}

func (this Position) String() string {
    return fmt.Sprintf("%s %.4f/%.4f, AvgPrice %.4f, Money %.4f ForceClose %.4f, lever %.0f",
        this.PType, this.AvailableAmount, this.Amount, this.AvgPrice, this.Money, this.ForceClosePrice, this.Leverage)
}

type Ticker struct {
    High       float64
    Low        float64
    Ask        float64
    Bid        float64
    Last       float64
    Vol        float64
    CreateTime time.Time
}

type Balance struct {
    //总保证金,总余额
    Amount float64
    //使用的保证金
    Deposit      float64
    RealProfit   float64
    UnrealProfit float64
    RiskRate     float64
    Currency     CurrencyUnit
}

func (this Balance) String() string {
    return fmt.Sprintf("BTC %.2f(%.2f), Profit %.2f %.2f, %.2f", this.Amount, this.Deposit,
        this.RealProfit, this.UnrealProfit, this.RiskRate)
}

type Kline struct {
    Id         string `column:"id"`
    Exname     string `column:"exname"`
    OpenTime   time.Time `column:"open_time"`
    CloseTime  time.Time `column:"close_time"`
    OpenPrice  float64 `column:"open_price"`
    ClosePrice float64 `column:"close_price"`
    HighPrice  float64 `column:"high_price"`
    LowPrice   float64 `column:"low_price"`
    Amount     float64 `column:"amount"`
    AvgPrice   float64 `column:"price"`
    Money      float64 `column:"money"`
    BuyNum     int `column:"buy_num"`
    SellNum    int `column:"sell_num"`
    TradeNum   int `column:"trade_num"`
    Fee        float64 `column:"fee"`
}

func NewKline(exname string, trade Trade, t time.Duration) *Kline {
    k := &Kline{
        Exname: exname,
        OpenTime: trade.CreateTime,
        CloseTime: trade.CreateTime.Add(t),
        OpenPrice: trade.Price,
        HighPrice: trade.Price,
        LowPrice: trade.Price,
    }
    k.AddTrade(trade)
    return k
}

func (k *Kline) AddTrade(t Trade) int {
    if k.OpenTime.Minute() > t.CreateTime.Minute() {
        return -1
    }
    if k.CloseTime.Minute() > t.CreateTime.Minute() {
        k.Amount += t.Amount
        k.Money += t.Amount * t.Price
        k.AvgPrice = k.Amount / k.Money
        if t.Price > k.HighPrice {
            k.HighPrice = t.Price
        }
        if t.Price < k.LowPrice {
            k.LowPrice = t.Price
        }
        k.ClosePrice = t.Price
        k.TradeNum += 1
        if t.TAction == Buy {
            k.BuyNum += 1
        } else {
            k.SellNum += 1
        }
        k.Fee += t.Fee
        return 0
    }
    return 1
}


