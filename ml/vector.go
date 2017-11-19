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

func NewVectorWithDense(dense []float64) *Vector {
    indices := make([]int, 0)
    values := make([]float64, 0)
    for i, v := range dense {
        if v != 0 {
            indices = append(indices, i)
            values = append(values, v)
        }
    }
    return NewVector(len(dense), indices, values)
}

func (v *Vector) Get(i int) float64 {
    return v.data[i]
}

func (v *Vector) DenseArray() []float64 {
    dense := make([]float64, 0, v.size)
    for i := 0; i < v.size; i++ {
        dense = append(dense, v.data[i])
    }
    return dense
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

func (v *Vector) DotDense(dense []float64) float64 {
    var dotSum float64
    for i, w := range v.data {
        dotSum += dense[i] * w
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

