package market

import (
    "net/http"
    "net/url"
    "fmt"
    "strings"
    "sort"
    "github.com/roydong/gmvc"
)

func CallRest(api string, query, data map[string]interface{}) (*http.Response, error) {
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
        query = append(query, fmt.Sprintf("%s=%v", k, v))
    }
    return strings.Join(query, "&")
}



func createSignature(params map[string]interface{}, skey string) string {
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

    return gmvc.MD5(strings.Join(sigs, "&"))
}


