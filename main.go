package main

import (
    "github.com/roydong/exchange/okcoin"
    "github.com/roydong/gtools"
    "log"
)

func main() {
    conf := gtools.NewTree()
    conf.LoadYamlFile("config", "poker/config/config.yml", true)

    okc := conf.Tree("config.market.okfuture")
    httpHost, _ := okc.String("http_host")
    apiKey, _ := okc.String("api_key")
    apiSecret, _ := okc.String("api_secret")
    ok := okcoin.NewOKFuture(httpHost, apiKey, apiSecret, "quater", 20)

    a, err := ok.GetTrades()
    log.Println(a, err)
}

