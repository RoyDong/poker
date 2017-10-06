package server

import (
    _ "net/http/pprof"
    "net/http"
    "log"
    "dw/poker/utils"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "runtime"
    "time"
    "dw/poker/context"
    "dw/poker/market"
    "fmt"
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
        http.ListenAndServe(conf.Server.PProfHost, nil)
    }()

    h := handler{
        sem: make(chan struct{}, conf.Server.MaxConn),
        timeout: time.Duration(conf.Server.MaxExecTime) * time.Millisecond,
    }
    log.Printf("start listen [%s]", conf.Server.Host)
    log.Fatal(http.ListenAndServe(conf.Server.Host, h))
}

type handler struct {
    sem chan struct{}
    timeout time.Duration
}

func (h handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
    h.sem <-struct{}{}
    done := make(chan struct{})
    var body []byte
    var code int
    go func() {
        defer func() {
            close(done)
            <-h.sem
        }()
        body, code = runHttpThread(req)
    }()
    select {
    case <-done:
        resp.WriteHeader(code)
        resp.Write(body)
    case <-time.After(h.timeout):
        resp.WriteHeader(http.StatusGatewayTimeout)
        resp.Write([]byte(fmt.Sprintf("reach max exec time %v", h.timeout)))
    }
}


