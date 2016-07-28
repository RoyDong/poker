package market

import (
    "container/list"
)

type averager struct {
    dataList *list.List
    dataMap map[int64]float64
    maxNum int
    total, avg float64
}



func newAverager(num int) *averager {
    ar := &averager{
        dataList: list.New(),
        dataMap: make(map[int64]float64, num),
        maxNum: num,
    }
    return ar
}

func (ar *averager) Add(idx int64, val float64) (bool, int64) {
    ar.dataMap[idx] = val
    ar.dataList.PushFront(idx)
    ar.total += val

    overflow := false
    if ar.dataList.Len() > ar.maxNum {
        el := ar.dataList.Back()
        idx, _ = el.Value.(int64)
        val = ar.dataMap[idx]
        ar.total -= val
        ar.dataList.Remove(el)
        delete(ar.dataMap, idx)
        overflow = true
    }

    ar.avg = ar.total / float64(ar.dataList.Len())
    return overflow, idx
}

func (ar *averager) CutTail(idx int64) {
    if val, has := ar.dataMap[idx]; has {
        el := ar.dataList.Back()
        i, _ := el.Value.(int64)
        if i == idx {
            ar.total -= val
            ar.dataList.Remove(el)
            delete(ar.dataMap, idx)
            ar.avg = ar.total / float64(ar.dataList.Len())
        } else {
            panic("data not sync")
        }
    }
}

func (ar *averager) Avg() float64 {
    return ar.avg
}

func (ar *averager) Len() int {
    return ar.dataList.Len()
}



