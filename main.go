package main

import (
    "log"
    "runtime"
    "github.com/roydong/poker/server"
)

func main() {
    log.SetFlags(log.Lshortfile | log.LstdFlags)
    runtime.GOMAXPROCS(runtime.NumCPU())
    server.RunHTTP("conf/config.yml")
}