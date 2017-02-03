package arbitrage

import (
    "container/list"
)

type averager struct {
    keys *list.List
    data map[int64]float64
    size int
    total, avg float64
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

    overflow := false
    if ar.keys.Len() > ar.size {
        el := ar.keys.Back()
        key, _ = el.Value.(int64)
        val = ar.data[key]
        ar.remove(key, val, el)
        overflow = true
    }

    ar.avg = ar.total / float64(ar.keys.Len())
    return overflow, key
}

func (ar *averager) AddPeek(top bool, key int64, val float64) (bool, int64) {
    if ar.Full() {
        for el := ar.keys.Back(); el != nil; el = el.Prev() {
            k, _ := el.Value.(int64)
            v := ar.data[k]
            if (top && val > v) || (!top && val < v){
                ar.remove(k, v, el)
                return ar.Add(key, val)
            }
        }
        return false, 0
    }
    return ar.Add(key, val)
}

func (ar *averager) CutTail(key int64) {
    if val, has := ar.data[key]; has {
        el := ar.keys.Back()
        k, _ := el.Value.(int64)
        if k == key {
            ar.remove(key, val, el)
            ar.avg = ar.total / float64(ar.keys.Len())
        } else {
            panic("data not sync")
        }
    }
}

func (ar *averager) remove(key int64, val float64, el *list.Element) {
    ar.total -= val
    ar.keys.Remove(el)
    delete(ar.data, key)
}

func (ar *averager) Avg() float64 {
    return ar.avg
}

func (ar *averager) Len() int {
    return ar.keys.Len()
}

func (ar *averager) Full() bool {
    return ar.keys.Len() >= ar.size
}



