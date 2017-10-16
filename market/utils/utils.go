package utils

import (
    "net/http"
    "net/url"
    "fmt"
    "strings"
    "sort"
    "io/ioutil"
    "crypto/md5"
    "io"
    "encoding/hex"
    "time"
    "errors"
)

var HTTPTimeout = time.Second

func CallRest(api string, query, data map[string]interface{}) ([]byte, error) {
    if query != nil {
        api = api + "?" + BuildHttpQuery(query)
    }

    var resp *http.Response
    var err error
    if data == nil {
        resp, err = http.Get(api)
    } else {
        form := url.Values{}
        for k, v := range data {
            form.Set(k, fmt.Sprintf("%v", v))
        }
        respond := make(chan bool)
        go func(){
            resp, err = http.PostForm(api, form)
            respond <- true
        }()
        select {
        case <- respond:
        case <- time.After(HTTPTimeout):
            err = errors.New(fmt.Sprintf("call api %s timeout %v", api, HTTPTimeout))
        }
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

