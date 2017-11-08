package main

import (
    "runtime"
    "io/ioutil"
    "log"
    "gopkg.in/yaml.v2"
    "net/http"
    "net"
    "golang.org/x/net/netutil"
    "dw/poker/context"
    "dw/poker/utils"
    "google.golang.org/grpc"
    "dw/poker/proto/exsync"
)

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    var conf = &context.Config{}
    yml, err := ioutil.ReadFile("conf/config.yml")
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
    err = utils.Init(conf)
    if err != nil {
        log.Fatalf("init utils fail . err[%s]", err.Error())
    }

    srvConf := conf.Exsync.Server
    logConf := conf.Exsync.Log
    utils.InitLog(logConf.LogDir, logConf.LogRotate, logConf.FilePrefix, srvConf.Debug)

    srv, err := newSyncService(conf)
    if err != nil {
        log.Fatalf("create service error %s", err.Error())
    }
    go func() {
        http.ListenAndServe(srvConf.PProfHost, nil)
    }()
    listener, err := net.Listen("tcp", srvConf.Host)
    if err != nil {
        log.Fatalf("failed to listen %v", srvConf.Host)
    }
    if srvConf.MaxConn > 0 {
        listener = netutil.LimitListener(listener, srvConf.MaxConn)
    }
    log.Printf("start listen [%s]", srvConf.Host)
    s := grpc.NewServer()
    exsync.RegisterSyncServiceServer(s, srv)
    s.Serve(listener)
}











