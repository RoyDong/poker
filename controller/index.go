package controller

import (
    "github.com/roydong/gmvc"
    "github.com/roydong/poker/service/market"
    "github.com/roydong/poker/arbitrage"
    "log"
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
        ticker.Buy, _ = gmvc.Store.Float("config.hedger.trade_amount")

        return r.JsonResponse(ticker)
    }, "/ticker")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewMarket("okcoin")

        b, c := okcoin.GetBalance()

        huobi := market.NewMarket("huobi")

        b1, c1 := huobi.GetBalance()

        market.NewOKFutureWS("this_week").GetBalance()

        return r.JsonResponse([]float64{b, c, b1, c1, c + c1})
    }, "/balance")


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        name, _ := r.String("name")
        huobi := market.NewMarket(name)

        id, _ := r.Int64("id")

        order := huobi.OrderInfo(id)

        return r.JsonResponse(order)

    }, "/order_info")



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

        hg := market.NewHedge(okcoin, huobi)
        hg.Start()

        return r.TextResponse("done")

    }, "/okcoin_huobi")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewMarket("okcoin")
        haobtc := market.NewMarket("haobtc")

        hg := market.NewHedge(okcoin, haobtc)
        hg.Start()

        return r.TextResponse("done")

    }, "/okcoin_haobtc")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        hg := market.NewHedge(market.NewMarket("okcoin"), market.NewMarket("okfuture_quarter"))
        hg.Start()

        return r.TextResponse("done")

    }, "/okcoin_future")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        week := arbitrage.NewExchange("okfuture_thisweek")
        quarter := arbitrage.NewExchange("okfuture_quarter")
        hg := arbitrage.NewHedge(week, quarter)
        hg.Start()

        return r.TextResponse("done")

    }, "/week_quarter")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        quarter := arbitrage.NewExchange("okfuture_quarter")
        lr := arbitrage.NewLeeksReaper(quarter)
        lr.Start()
        return r.TextResponse("done")
    }, "/okfuture_leeksreaper")


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        //a := arbitrage.NewExchange("okfuture_thisweek")
        a := arbitrage.NewOKFuture("quarter", 20)

        gmvc.Store.Set("aa", a)

        a.GetBalance()

        return r.TextResponse("done")

    }, "/okfuture")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        a, _ := gmvc.Store.Get("aa").(*arbitrage.OKFuture)

        amount, _ := r.Float("amount")
        typ, _ := r.Int("type")

        id := a.Trade(typ, amount, 0)

        return r.TextResponse(fmt.Sprintf("id: %v", id))

    }, "/open")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        a, _ := gmvc.Store.Get("aa").(*market.OKFutureWS)

        id, _ := r.Int64("id")

        rs := a.CancelOrder(id)
        log.Println(rs)

        return r.TextResponse("done")

    }, "/cancel")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        a, _ := gmvc.Store.Get("aa").(*arbitrage.OKFuture)

        id, _ := r.Int64("id")

        rs := a.Order(id)

        log.Println(rs)
        return r.TextResponse("done")

    }, "/order")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        lr := arbitrage.NewLeeksReaper(arbitrage.NewExchange("okfuture_quarter"))
        go lr.Start()
        return r.TextResponse("done")

    }, "/leeks_reaper")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        huobi := market.NewHuobiWS()

        huobi.Connect()

        return r.TextResponse("done")

    }, "/fix_test")


    gmvc.WSActionMap["ws"] = func(wsm *gmvc.WSMessage) {
        val, _ := wsm.String("a")

        q := map[string]interface{}{
            "aaa": "sss",
            "value": val,
        }
        wsm.Send("name", q, nil)
    }
}

