package utils

import (
    "net/http"
    "fmt"
    "strings"
    "sort"
    "io/ioutil"
    "crypto/md5"
    "io"
    "encoding/hex"
    "time"
    "errors"
    "crypto/sha256"
    "crypto/hmac"
)

var HTTPTimeout = time.Second

func ReqHttp(host string, query, post, header map[string]interface{}) ([]byte, error) {
    if len(query) > 0 {
        host = host + "?" + BuildHttpQuery(query)
    }
    var req *http.Request
    var err error
    if len(post) > 0 {
        req, err = http.NewRequest("POST", host, strings.NewReader(BuildHttpQuery(post)))
    } else {
        req, err = http.NewRequest("GET", host, nil)
    }
    if err != nil {
        return nil, err
    }
    if req.Method == "POST"  {
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    }
    if len(header) > 0 {
        for k, v := range header {
            req.Header.Add(k, fmt.Sprintf("%v", v))
        }
    }

    var resp *http.Response
    done := make(chan struct{})
    go func(){
        resp, err = http.DefaultClient.Do(req)
        close(done)
    }()
    select {
    case <- done:
    case <- time.After(HTTPTimeout):
        err = errors.New(fmt.Sprintf("call api %s timeout %v", host, HTTPTimeout))
    }
    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    return body, err
}

func BuildHttpQuery(data map[string]interface{}) string {
    query := make([]string, 0, len(data))
    for k, v := range data {
        query = append(query, fmt.Sprintf("%s=%v", k, v))
    }
    return strings.Join(query, "&")
}

func CreateSignature(params map[string]interface{}, skey string) string {
    keys := make([]string, 0, len(params))
    for k := range params {
        keys = append(keys, k)
    }

    sort.Strings(keys)

    sigs := make([]string, 0, len(keys))
    for _, k := range keys {
        sigs = append(sigs, fmt.Sprintf("%s=%v", k, params[k]))
    }

    sigs = append(sigs, "secret_key=" + skey)

    return MD5(strings.Join(sigs, "&"))
}

func MD5(txt string) string {
    hash := md5.New()
    if _, err := io.WriteString(hash, txt); err != nil {
        panic(err)
    }
    return hex.EncodeToString(hash.Sum(nil))
}

func SHA256(txt string) string {
    hash := sha256.New()
    if _, err := io.WriteString(hash, txt); err != nil {
        panic(err)
    }
    return hex.EncodeToString(hash.Sum(nil))
}

func HMAC_SHA256(k, v string) string {
    hash := hmac.New(sha256.New, []byte(k))
    if _, err := io.WriteString(hash, v); err != nil {
        panic(err)
    }
    return hex.EncodeToString(hash.Sum(nil))
}





