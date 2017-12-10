package gamble

import (
    "dw/poker/market"
    "time"
    "dw/poker/utils"
    "log"
)


type SpreadMargin struct {
    tradeAmount float64
    margin float64
}

func (sm *SpreadMargin) Run(ex *market.Exchange) error {
    for {
        time.Sleep(10 * time.Second)
        err := ex.CancelAllOrders()
        if err != nil {
            utils.DebugLog.Write(err.Error())
            continue
        }

        err = ex.WaitOrders()
        if err != nil {
            utils.DebugLog.Write(err.Error())
            continue
        }

        asks, bids, err := ex.GetDepth()
        log.Println(asks, bids)
        if err != nil {
            utils.DebugLog.Write(err.Error())
            continue
        }
        if len(asks) == 0 || len(bids) == 0 {
            continue
        }

        long, short ,err := ex.GetPosition()
        sellPrice := asks[0].Price * (1 + sm.margin)
        buyPrice := bids[0].Price * (1 - sm.margin)
        utils.DebugLog.Write("sellPrice %.6f buyPrice %.6f long %.0f short %.0f",
            sellPrice, buyPrice, long.GetAmount(), short.GetAmount())

        if short.GetAmount() > 0 {
            ex.CloseShort(short.GetAvailableAmount(), buyPrice)
        } else {
            ex.OpenShort(sm.tradeAmount, sellPrice)
        }
        if long.GetAmount() > 0 {
            ex.CloseLong(long.GetAvailableAmount(), sellPrice)
        } else {
            ex.OpenLong(sm.tradeAmount, buyPrice)
        }
    }
}

