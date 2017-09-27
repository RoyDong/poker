package server

import (
    "net/http"
    "log"
    "github.com/roydong/poker/utils"
    "github.com/roydong/poker/common"
    "gopkg.in/yaml.v2"
    "io/ioutil"
)


func RunHTTP(filename string) {
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

    http.HandleFunc("/", dispatch)

    log.Printf("start server port [%s]", conf.Server.Host)
    err = http.ListenAndServe(conf.Server.Host, nil)
    if err != nil {
        log.Fatalf("start server fail . err[%s]", err.Error())
    }
    log.Printf("start server success . [%s]", conf.Server.Host)
}


