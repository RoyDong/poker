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
        okcoin := market.NewOKCoin()

        v := okcoin.Ticker()

        huobi := market.NewHuobi()

        h := huobi.Ticker()


        return r.JsonResponse([]float64{v.Last, h.Last, v.Last - h.Last})
    }, "/json")


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        okcoin := market.NewOKCoinCom()

        v := okcoin.Ticker()

        i := okcoin.Index()

        p := (v.Last - i) / i

        return r.JsonResponse([]float64{v.Last, i, gmvc.Round(p, 4) * 100})

    }, "/okcoin_premium")

    gmvc.WSActionMap["ws"] = func(wsm *gmvc.WSMessage) {
        val, _ := wsm.String("a")

        q := map[string]interface{}{
            "aaa": "sss",
            "value": val,
        }
        wsm.Send("name", q, nil)
    }
}

