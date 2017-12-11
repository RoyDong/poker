package gamble

import (
    "dw/poker/market"
    "time"
    "dw/poker/utils"
    "log"
)


type GapMargin struct {
    tradeAmount float64
    margin float64
}

func (m *GapMargin) Run(ex *market.Exchange) error {
    for {
        log.Println("new round")
        time.Sleep(10 * time.Second)
        err := ex.CancelAllOrders()
        if err != nil {
            utils.DebugLog.Write(err.Error())
            continue
        }

        err = ex.WaitOrdersDone()
        if err != nil {
            utils.DebugLog.Write(err.Error())
            continue
        }

        asks, bids, err := ex.GetDepth()
        if err != nil {
            utils.DebugLog.Write("gp get depth error", err.Error())
            continue
        }
        if len(asks) == 0 || len(bids) == 0 {
            utils.DebugLog.Write("empty depth")
            continue
        }

        long, short ,err := ex.GetPosition()
        sellPrice := asks[0].Price * (1 + m.margin)
        buyPrice := bids[0].Price * (1 - m.margin)
        utils.DebugLog.Write("sellPrice %.6f buyPrice %.6f long %.0f short %.0f",
            sellPrice, buyPrice, long.GetAmount(), short.GetAmount())

        if short.GetAmount() > 0 {
            ex.CloseShort(short.GetAvailableAmount(), buyPrice)
            utils.DebugLog.Write("close short %.0f %.6f", short.GetAvailableAmount(), buyPrice)
        } else {
            ex.OpenShort(m.tradeAmount, sellPrice)
            utils.DebugLog.Write("open short %.0f %.6f", m.tradeAmount, sellPrice)
        }
        if long.GetAmount() > 0 {
            ex.CloseLong(long.GetAvailableAmount(), sellPrice)
            utils.DebugLog.Write("close long %.0f %.6f", long.GetAvailableAmount(), sellPrice)
        } else {
            ex.OpenLong(m.tradeAmount, buyPrice)
            utils.DebugLog.Write("open long %.0f %.6f", m.tradeAmount, buyPrice)
        }
    }
}

