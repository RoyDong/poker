package ml

import (
    "fmt"
    "sort"
)

type Vector struct {
    size int
    indices []int
    values []float64
    data map[int]float64
}

func NewVector(size int, indices []int, values []float64) *Vector {
    if len(indices) != len(values) {
        panic("indece not match values")
    }
    valmap := make(map[int]float64, len(indices))
    for i := 0; i < len(indices); i++ {
        valmap[indices[i]] = values[i]
    }
    vec := &Vector{
        size: size,
        indices:indices,
        values:values,
        data:valmap,
    }
    return vec
}

func (v *Vector) Get(i int) float64 {
    return v.data[i]
}

func (v *Vector) Array() []float64 {

}

func (v *Vector) Size() int {
    return v.size
}

func (v *Vector) Indices() []int {
    return v.indices
}

func (v *Vector) Dot(vec *Vector) float64 {
    var dotSum float64
    for i, w := range vec.data {
        dotSum += v.data[i] * w
    }
    return dotSum
}

func (v *Vector) String() string {
    out := make([]int, 0, len(v.data))
    for k := range v.data {
        out = append(out, k)
    }
    sort.Ints(out)
    return fmt.Sprintf("%d %v", v.size, out)
}


type LabeledPoint struct {
    Label float64
    Features *Vector
}

