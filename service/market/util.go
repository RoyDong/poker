package market

import (
    "net/http"
    "net/url"
    "fmt"
    "strings"
    "sort"
    "github.com/roydong/gmvc"
    "io/ioutil"
)

func CallRest(api string, query, data map[string]interface{}) *gmvc.Tree {
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
        resp, err = http.PostForm(api, form)
    }
    if err != nil {
        gmvc.Logger.Println("call " + api + " error")
        return nil
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        gmvc.Logger.Println("call " + api + " error")
        return nil
    }

    tree := gmvc.NewTree()
    if err = tree.LoadJson("", body, false); err != nil {
        gmvc.Logger.Println("call " + api + " error not json")
        return nil
    }

    return tree
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


/*
根据深度获得需要卖出amount数量的btc需要的价位,即获取对应的买单价(bid price)
 */
func GetSellPrice(amount float64, bids [][]float64) float64 {
    var sum, price float64
    for _, bid := range bids {
        price = bid[0]
        sum += bid[1]
        if sum >= amount {
            break
        }
    }
    return price
}

/*
根据深度获得需要买入amount数量的btc需要的价位,即获取对应的卖单价(ask price)
 */
func GetBuyPrice(amount float64, asks [][]float64) float64 {
    var sum, price float64
    for _, ask := range asks {
        price = ask[0]
        sum += ask[1]
        if sum >= amount {
            break
        }
    }
    return price
}
