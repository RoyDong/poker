package server

import (
    _ "net/http/pprof"
    "net/http"
    "log"
    "dw/poker/utils"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "runtime"
    "dw/poker/context"
    "dw/poker/market"
    "net"
    "golang.org/x/net/netutil"
)


func Run(filename string) {
    runtime.GOMAXPROCS(runtime.NumCPU())

    var conf = &context.Config{}
    yml, err := ioutil.ReadFile(filename)
    if err != nil {
        log.Fatalf("error read yml config %s", err.Error())
    }
    err = yaml.Unmarshal(yml, conf)
    if err != nil {
        log.Fatalf("error parse yml config  [%s]", err.Error())
    }
    if err != nil {
        log.Fatalf("init config fail . err[%s]", err.Error())
    }

    market.Init(conf)

    //init log
    utils.Init(conf)
    err = initRouter(conf)
    if err != nil {
        log.Fatalf("init router fail . err[%s]", err.Error())
    }

    go func() {
        http.ListenAndServe(conf.Server.PProfHost, nil)
    }()
    listener, err := net.Listen("tcp", conf.Server.Host)
    if err != nil {
        log.Fatalf("failed to listen %v", conf.Server.Host)
    }
    if conf.Server.MaxConn > 0 {
        listener = netutil.LimitListener(listener, conf.Server.MaxConn)
    }
    log.Printf("start listen [%s]", conf.Server.Host)
    log.Fatal(http.Serve(listener, handler{}))
}

