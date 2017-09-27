package server

import (
    "github.com/roydong/poker/common"
    "github.com/roydong/poker/utils"
    "net/http"
    "strings"
    "github.com/roydong/poker/module/market"
    "github.com/roydong/poker/module/riskctrl"
)

type IModule interface {
    Init(conf *common.Config) error
    Run(ctx *common.Context) error
}

var routes = map[string][]IModule{
    "/market": {
        &market.Market{},
    },
    "/riskctrl": {
        &riskctrl.RiskCtrl{},
    },
}


func initRouter(conf *common.Config) error {
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

func initContext(route string) *common.Context {
    return &common.Context{}
}

var defaultRoute = "/"

func dispatch(resp http.ResponseWriter, req *http.Request) {
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

