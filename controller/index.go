package controller

import (
    "github.com/roydong/gmvc"
    "github.com/roydong/poker/service/market"
)

func init() {
    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {

        name, _ := r.String("$1")



        sess := r.Session

        sess.Set("aaa", name)



        return r.HtmlResponse(name, "index/index")

    }, `/home/(\w+)`)


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewMarket("okcoin")

        v := okcoin.LastTicker()

        huobi := market.NewMarket("huobi")

        h := huobi.LastTicker()


        return r.JsonResponse([]float64{v.Last, h.Last, v.Last - h.Last})
    }, "/json")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewMarket("okcoin")

        b, c := okcoin.GetBalance()

        huobi := market.NewMarket("huobi")

        b1, c1 := huobi.GetBalance()

        return r.JsonResponse([]float64{b, c, b1, c1, c + c1})
    }, "/balance")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        huobi := market.NewMarket("huobi")

        huobi.Sell(0.02)
        huobi.Buy(50)

        return r.TextResponse("done")

    }, "/trade_hb")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {

        tree := gmvc.NewTree()
        tree.LoadJson("", []byte(`{"code":1,"msg":"服务器繁忙","message":"服务器繁忙"}`), false)

        gmvc.Logger.Println(tree.Float64("code"))

        return r.TextResponse("aa")

    }, "/trade_ok")


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewOKCoinCom()

        v := okcoin.Ticker()

        i := okcoin.Index()

        p := (v.Last - i) / i

        return r.JsonResponse([]float64{v.Last, i, gmvc.Round(p, 4) * 100})

    }, "/okcoin_premium")

    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewMarket("okcoin")

        huobi := market.NewMarket("huobi")

        hg := market.NewHedger(okcoin, huobi)
        hg.Start()

        return r.TextResponse("done")

    }, "/sync_ticker")


    gmvc.WSActionMap["ws"] = func(wsm *gmvc.WSMessage) {
        val, _ := wsm.String("a")

        q := map[string]interface{}{
            "aaa": "sss",
            "value": val,
        }
        wsm.Send("name", q, nil)
    }
}

