package server

import (
    _ "net/http/pprof"
    "net/http"
    "log"
    "github.com/roydong/poker/utils"
    "github.com/roydong/poker/common"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "runtime"
    "net"
    "golang.org/x/net/netutil"
    "time"
)


func Run(filename string) {
    runtime.GOMAXPROCS(runtime.NumCPU())

    var conf = &common.Config{}
    yml, err := ioutil.ReadFile(filename)
    if err != nil {
        log.Fatalf("error read yml config")
    }
    err = yaml.Unmarshal(yml, conf)
    if err != nil {
        log.Fatalf("error parse yml config  [%s]", err.Error())
    }
    if err != nil {
        log.Fatalf("init global conf fail . err[%s]", err.Error())
    }

    //init log
    utils.Init(conf)
    err = initRouter(conf)
    if err != nil {
        log.Fatalf("init router fail . err[%s]", err.Error())
    }

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


