package controller

import (
    "github.com/roydong/gmvc"
    "github.com/roydong/poker/service/market"
    "fmt"
)

func init() {
    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {

        name, _ := r.String("$1")



        sess := r.Session

        sess.Set("aaa", name)



        return r.HtmlResponse(name, "index/index")

    }, `/home/(\w+)`)


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        huobi := market.NewMarket("huobi")
        okcoin := market.NewMarket("okcoin")
        haobtc := market.NewMarket("haobtc")
        okfut := market.NewMarket("okfuture")

        oka, okb := okcoin.GetDepth()
        hba, hbb := huobi.GetDepth()

        hta, htb := haobtc.GetDepth()

        ofa, ofb := okfut.GetDepth()

        a := map[string][][]float64{
            "ok_ask": oka,
            "ok_bid": okb,

            "hb_ask": hba,
            "hb_bid": hbb,

            "ht_ask": hta,
            "ht_bid": htb,

            "of_ask": ofa,
            "of_bid": ofb,
        }

        return r.JsonResponse(a)
    }, "/depth")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        haobtc := market.NewMarket("huobi")

        ticker := haobtc.LastTicker()

        conf,_ := gmvc.Store.Int64("config.hedger.trade_amount")

        ticker.Time = conf
        ticker.Buy, _ = gmvc.Store.Float64("config.hedger.trade_amount")

        return r.JsonResponse(ticker)
    }, "/ticker")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewMarket("okcoin")

        b, c := okcoin.GetBalance()

        huobi := market.NewMarket("huobi")

        b1, c1 := huobi.GetBalance()

        return r.JsonResponse([]float64{b, c, b1, c1, c + c1})
    }, "/balance")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        huobi := market.NewMarket("huobi")

        //id := huobi.Buy(50)
        id := huobi.Sell(0.01)

        return r.TextResponse(fmt.Sprintf("order_id: %v", id))

    }, "/trade_hb")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        name, _ := r.String("name")
        huobi := market.NewMarket(name)

        id, _ := r.Int64("id")

        order := huobi.OrderInfo(id)

        return r.JsonResponse(order)

    }, "/order_info")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        haobtc := market.NewMarket("haobtc")

        haobtc.Sell(0.02)
        haobtc.Buy(50)

        return r.TextResponse("done")

    }, "/trade_ht")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {

        ok := market.NewMarket("okcoin")

        //id := ok.Buy(50)
        id := ok.Sell(0.01)

        return r.JsonResponse(id)

    }, "/trade_ok")


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        ok := market.NewOKFuture("quarter")

        v := ok.LastTicker()

        i := ok.Index()

        p := (v.Last - i) / i

        return r.JsonResponse([]float64{v.Last, i, gmvc.Round(p, 4) * 100})

    }, "/okcoin_premium")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewMarket("okcoin")
        huobi := market.NewMarket("huobi")

        hg := market.NewHedger(okcoin, huobi)
        hg.Start()

        return r.TextResponse("done")

    }, "/okcoin_huobi")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewMarket("okcoin")
        haobtc := market.NewMarket("haobtc")

        hg := market.NewHedger(okcoin, haobtc)
        hg.Start()

        return r.TextResponse("done")

    }, "/okcoin_haobtc")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        hg := market.NewHedger(market.NewMarket("okcoin"), market.NewMarket("okfuture_quarter"))
        hg.Start()

        return r.TextResponse("done")

    }, "/okcoin_future")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        week := market.NewMarket("okfuture_thisweek")
        quarter := market.NewMarket("okfuture_quarter")
        hg := market.NewHedger(week, quarter)
        hg.Start()

        return r.TextResponse("done")

    }, "/week_quarter")


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        huobi := market.NewHuobi()
        huobi.WSConnect()

        return r.TextResponse("done")

    }, "/hb_ws")


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        a := market.NewOKFutureWS("this_week")

        gmvc.Store.Set("aa", a)

        return r.TextResponse("done")

    }, "/okfuture")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        a, _ := gmvc.Store.Get("aa").(*market.OKFutureWS)

        a.RemoveChannels()

        return r.TextResponse("done")

    }, "/remove_channel")


    gmvc.WSActionMap["ws"] = func(wsm *gmvc.WSMessage) {
        val, _ := wsm.String("a")

        q := map[string]interface{}{
            "aaa": "sss",
            "value": val,
        }
        wsm.Send("name", q, nil)
    }
}

