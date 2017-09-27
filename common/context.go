package common

import "net/http"

type Context struct {

    Request *http.Request

    RespBody []byte
}

