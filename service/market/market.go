package market

import (
    "net/http"
    "fmt"
    "strings"
    "net/url"
    "sort"
    "github.com/roydong/gmvc"
)

func CallRest(api string, query map[string]interface{}, data map[string]interface{}) (*http.Response, error) {
    if query != nil {
        api = api + "?" + BuildHttpQuery(query)
    }

    if data == nil {
        return http.Get(api)
    }

    form := url.Values{}
    for k, v := range data {
        form.Set(k, fmt.Sprintf("%v", v))
    }

    return http.PostForm(api, form)
}

func BuildHttpQuery(data map[string]interface{}) string {
    query := make([]string, 0, len(data))
    for k, v := range data {
        query = append(query, fmt.Sprintf("%v=%v", k, v))
    }
    return strings.Join(query, "&")
}



func createSignature(params map[string]interface{}) string {
    var keys []string

    for k := range params {
        keys = append(keys, k)
    }

    sort.Strings(keys)

    sigs := make([]string, 0, len(keys))
    for _, k := range keys {
        sigs = append(sigs, fmt.Sprintf("%v=%v", k, params[k]))
    }

    return gmvc.MD5(strings.Join(sigs, "&"))
}


type Ticker struct {
    High float64
    Low  float64
    Sell float64
    Buy  float64
    Last float64
    Vol  float64
    Time int64
}
