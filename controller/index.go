package controller

import "github.com/roydong/gmvc"

func init()  {
    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {

        name, _ := r.String("$1")



        sess := r.Session

        sess.Set("aaa", name)



        return r.HtmlResponse(name, "index/index")

    }, `/home/(\w+)`)


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {

        val, _ := r.Session.Get("aaa").Str()

        data := map[string]string{
            "aaa": "aaa",
            "name": "Roy",
            "sex": "男",
            "sess": val,
        }
        return r.JsonResponse(data)
    }, "/json")


    gmvc.WSActionMap["ws"] = func(wsm *gmvc.WSMessage) {
        val, _ := wsm.String("a")

        q := map[string]interface{}{
            "aaa": "sss",
            "value": val,
        }
        wsm.Send("name", q, nil)
    }
}

