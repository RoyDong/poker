package server

import (
    "github.com/roydong/poker/context"
    "github.com/roydong/poker/utils"
    "github.com/roydong/poker/module/market"
    "github.com/roydong/poker/module/riskctrl"
    "net/http"
    "strings"
)

type IModule interface {
    Init(conf *context.Config) error
    Run(ctx *context.Context) error
}

var routes = map[string][]IModule{
    "/market": {
        &market.Market{},
    },
    "/riskctrl": {
        &riskctrl.RiskCtrl{},
    },
}


func initRouter(conf *context.Config) error {
    for path, modules := range routes {
        for i := 0; i < len(modules); i++ {
            err := modules[i].Init(conf)
            if err != nil {
                utils.FatalLog.Write("init modules[%s/%d] fail. err[%s]", path, i, err.Error())
                return err
            }
        }
    }
    return nil
}

func initContext(route string) *context.Context {
    return &context.Context{}
}

var defaultRoute = "/"

type handler string

func (h handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
    path := strings.ToLower(strings.TrimSpace(req.URL.Path))
    modules := routes[path]
    if len(modules) <= 0 && len(defaultRoute) > 0 {
        modules = routes[defaultRoute]
    }
    if len(modules) <= 0 {
        utils.WarningLog.Write("http not found[%s]", req.URL.Path)
        resp.WriteHeader(http.StatusNotFound)
        resp.Write([]byte("404 not found"))
        return
    }
    ctx := initContext(path)
    ctx.Request = req
    for i := 0; i < len(modules); i++ {
        err := modules[i].Run(ctx)
        if err != nil {
            utils.FatalLog.Write("run modules[%s/%d] fail. err[%s]", path, i, err.Error())
            resp.Write([]byte(err.Error()))
            return
        }
    }
    resp.Write(ctx.RespBody)
}

