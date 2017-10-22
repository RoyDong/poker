package bitmex

import (
    "dw/poker/market/utils"
    putils "dw/poker/utils"
)

type Xbtusd struct {
    apiKey string
    apiSecret string
    ws *utils.WsClient
}

func NewXbtusd(apiKey, apiSecret, wss, host string) (*Xbtusd, error) {
    var err error
    xbt := &Xbtusd{}
    xbt.apiKey = apiKey
    xbt.apiSecret = apiSecret
    xbt.ws, err = utils.NewWsClient(wss, host, xbt.newMsg, xbt.connected)

    return xbt, err
}


func (xbt *Xbtusd) newMsg(msg []byte) {
    putils.DebugLog.Write(string(msg))
}

type wscmd struct {
    Op string
    Args []string
}

func newWscmd(op string, args []string) wscmd {
    return wscmd{op, args}
}

func (xbt *Xbtusd) connected() {
    nonce := "poker"
    cmd := wscmd{"auth", []string{xbt.apiKey, nonce, xbt.sign(nonce)}}
    putils.DebugLog.Write("%v", cmd)
    xbt.ws.SendJson(cmd)
}

func (xbt *Xbtusd) sign(nonce string) string {
    return utils.HMAC_SHA256(xbt.apiSecret, "GET/realtime" + nonce)
}

