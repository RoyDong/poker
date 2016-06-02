package controller

import "github.com/roydong/gmvc"

func init()  {
    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {

        name, _ := r.String("$1")
        return r.HtmlResponse(name, "index/index")

    }, `/home/(\w+)`)


    gmvc.SetAction(func(r *gmvc.Request) *gmvc.Response {
        data := map[string]string{
            "aaa": "aaa",
            "name": "Roy",
            "sex": "男",
        }
        return r.JsonResponse(data)
    }, "/json")
}

