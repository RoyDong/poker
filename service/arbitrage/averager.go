package arbitrage

import (
"container/list"
"math"
)

type averager struct {
    keys *list.List
    data map[int64]float64
    size int
    total, avg float64
    minKey, maxKey int64
}

func newAverager(size int) *averager {
    ar := &averager{
        keys: list.New(),
        data: make(map[int64]float64, size),
        size: size,
    }
    return ar
}

func (ar *averager) Add(key int64, val float64) (bool, int64) {
    ar.data[key] = val
    ar.keys.PushFront(key)
    ar.total += val

    if ar.keys.Len() == 1 {
        ar.minKey = key
        ar.maxKey = key
    }

    var k int64
    overflow := false
    if ar.keys.Len() > ar.size {
        el := ar.keys.Back()
        ar.keys.Remove(el)
        overflow = true

        k, _ = el.Value.(int64)
        ar.total -= ar.data[k]
    }

    if val < ar.Min() {
        ar.minKey = key
    }
    if val > ar.Max() {
        ar.maxKey = key
    }

    ar.avg = ar.total / float64(ar.keys.Len())
    return overflow, k
}

func (ar *averager) remove(key int64, val float64, el *list.Element) {
    ar.total -= val
    ar.keys.Remove(el)
    delete(ar.data, key)
}

func (ar *averager) Avg() float64 {
    return ar.avg
}

func (ar *averager) Mid() float64 {
    return (ar.Min() + ar.Max()) / 2
}

func (ar *averager) Min() float64 {
    if v, has := ar.data[ar.minKey]; has {
        return v
    }
    var min = math.Inf(1)
    var minKey int64
    for el := ar.keys.Back(); el != nil; el = el.Prev() {
        k, _ := el.Value.(int64)
        v := ar.data[k]
        if v < min {
            minKey = k
            min = v
        }
    }
    ar.minKey = minKey
    return min
}

func (ar *averager) Max() float64 {
    if v, has := ar.data[ar.maxKey]; has {
        return v
    }
    var max = math.Inf(-1)
    var maxKey int64
    for el := ar.keys.Back(); el != nil; el = el.Prev() {
        k, _ := el.Value.(int64)
        v := ar.data[k]
        if v > max {
            maxKey = k
            max = v
        }
    }
    ar.maxKey = maxKey
    return max
}

func (ar *averager) Len() int {
    return ar.keys.Len()
}

func (ar *averager) Full() bool {
    return ar.keys.Len() >= ar.size
}


