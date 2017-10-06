package server

import (
    _ "net/http/pprof"
    "net/http"
    "log"
    "dw/poker/utils"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "runtime"
    "net"
    "golang.org/x/net/netutil"
    "time"
    "dw/poker/context"
    "dw/poker/market"
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

    //init log
    utils.Init(conf)
    err = initRouter(conf)
    if err != nil {
        log.Fatalf("init router fail . err[%s]", err.Error())
    }

    market.Init(conf)

    go func() {
        log.Println(http.ListenAndServe(conf.Server.PProfHost, nil))
    }()

    lis, err := net.Listen("tcp", conf.Server.Host)
    if err != nil {
        log.Fatalf("listen to port failed [%s]", err.Error())
    }
    if conf.Server.MaxConn > 0 {
        lis = netutil.LimitListener(lis, conf.Server.MaxConn)
    }

    h := http.Handler(handler("http"))
    if conf.Server.Timeout > 0 {
        timeout := time.Duration(conf.Server.Timeout) * time.Millisecond
        h = http.TimeoutHandler(h, timeout, "")
    }

    log.Printf("start listen [%s]", conf.Server.Host)
    log.Fatal(http.Serve(lis, h))
}


