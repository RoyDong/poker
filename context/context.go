package context

import "net/http"

type Context struct {

    Request *http.Request

    RespBody []byte
}

