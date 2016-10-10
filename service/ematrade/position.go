package ematrade

import "time"

const (
    TypeLong = 1
    TypeShort = 2

    StateOpening = 1
    StateOpen    = 2
    StateClosing = 3
    StateClose   = 4
)

type position struct {
    amount, price float64

    postionType, state int

    closeAmount, closePrice float64

    //平仓成交数据
    closeData []float64

    profit float64

    openAt, closeAt time.Time
}

func openLong(amount, price float64) *position {

}

func openClose(amount, price float64) *position {

}

func (pos *position)
