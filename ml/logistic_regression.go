package ml

import (
    "math"
    "strings"
    "strconv"
    "errors"
    "sync"
    "math/rand"
    "dw/poker/utils"
    "log"
    "io/ioutil"
)

type LogisticRegression struct {
    *utils.FileLoader
    version  string
    mu sync.RWMutex
    weights  *Vector
}

func (m *LogisticRegression) GetWeights() *Vector {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.weights
}

func (m *LogisticRegression) Version() string {
    return m.version
}

func (m *LogisticRegression) Predict(vec *Vector) float64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if m.weights.Size() == vec.Size() {
        dotSum := m.weights.Dot(vec)
        return 1.0 / (1.0 + math.Exp(-dotSum))
    }
    return 0
}

func (m *LogisticRegression) PredictWithRandWeight(vec *Vector,
    i int, min, max float64) (float64, float64) {

    m.mu.RLock()
    defer m.mu.RUnlock()
    if m.weights.Size() == vec.Size() {
        dotSum := m.weights.Dot(vec)

        //如果i位置权重为0，随机一个权重
        var w float64
        if m.weights.Get(i) == 0 {
            w = (min + rand.Float64() * (max - min)) * vec.Get(i)
            dotSum += w
        }
        return 1.0 / (1.0 + math.Exp(-dotSum)), w
    }
    return 0, 0
}

func (m *LogisticRegression) LoadFromFile(file string) error {
    raw, err := ioutil.ReadFile(file)
    if err != nil {
        return err
    }
    return m.newContent(raw)
}

func (m *LogisticRegression) newContent(bytes []byte) error {
    var err error
    var headBody = strings.Split(string(bytes), "\r\n")
    if len(headBody) != 2 {
        err = errors.New("cpc/ml: model data malformed")
        utils.FatalLog.Write(err.Error())
        return err
    }

    var size int64
    if rows := strings.Split(headBody[0], "\n"); len(rows) > 0 {
        for _, row := range rows {
            vals := strings.SplitN(strings.TrimSpace(row), " ", 2)
            if len(vals) == 2 {
                k := strings.TrimSpace(vals[0])
                v := strings.TrimSpace(vals[1])
                if len(k) > 0 {
                    switch k {
                    case "num_features":
                        size, err = strconv.ParseInt(v, 10, 64)
                        if err != nil {
                            utils.FatalLog.Write("cpc/ml: bad model data, num_features must be integer")
                            return err
                        }

                        //TODO
                    case "version":
                    case "num_classes":
                    case "model_path":
                    case "date":
                    case "auprc":
                    case "aur":
                    }
                }
            }
        }
    } else {
        err = errors.New("cpc/ml: empty head")
        utils.FatalLog.Write(err.Error())
        return err
    }

    indices := make([]int, 0, size)
    values := make([]float64, 0, size)
    if rows := strings.Split(headBody[1], "\n"); len(rows) > 0 {
        for _, row := range rows {
            vals := strings.SplitN(strings.TrimSpace(row), " ", 2)
            if len(vals) == 2 {
                k := strings.TrimSpace(vals[0])
                v := strings.TrimSpace(vals[1])
                if len(k) > 0 {
                    i, err := strconv.ParseInt(k, 10, 64)
                    if err != nil {
                        utils.FatalLog.Write("cpc/ml: bad model data, index must be integer [%s, %s]", k, v)
                        return err
                    }
                    v, err := strconv.ParseFloat(v, 64)
                    if err != nil {
                        utils.FatalLog.Write("cpc/ml: bad model data, weight must be float64")
                        return err
                    }
                    indices = append(indices, int(i))
                    values = append(values, v)
                }
            }
        }
        m.mu.Lock()
        m.weights = NewVector(int(size), indices, values)
        m.mu.Unlock()
        utils.WarningLog.Write("cpc/ml LR model loaded %s %d", m.version, len(indices))
    } else {
        err = errors.New("cpc/ml: empty body")
        utils.FatalLog.Write(err.Error())
        return err
    }

    return nil
}

func (m *LogisticRegression) Train(sample []*LabeledPoint) {
}

func (m *LogisticRegression) initWeights(size int) *Vector {
    return NewVector(size, []int{}, []float64{})
}

func (m *LogisticRegression) StocGradAscent(sample []*LabeledPoint, iterNum int) {
    first := sample[0]
    log.Println(first.Features.Size(), first.Features.DenseArray(), len(sample))
    weights := make([]float64, first.Features.Size())
    alpha := 0.01
    l1norm := 0.0
    for iter := 0; iter < iterNum; iter++ {
        var minloss float64
        pnum := 0
        for _, lp := range sample {
            if lp.Label > 0 {
                pnum++
            }
            dotSum := lp.Features.DotDense(weights)
            hypothesis := 1.0 / (1.0 + math.Exp(-dotSum))

            loss := hypothesis - lp.Label
            minloss = loss
            for _, i := range lp.Features.Indices() {
                v := lp.Features.Get(i)
                weights[i] = weights[i] - alpha * (loss * v + weights[i] * l1norm)
            }
        }
        utils.DebugLog.Write("SGD iter %d: loss %.6f %v", iter, minloss, pnum)
    }
    log.Println(weights)
    m.weights = NewVectorWithDense(weights)
}

func (m *LogisticRegression) BatchGradAscent(sample []*LabeledPoint, iterNum int) {
    first := sample[0]
    log.Println(first.Features.Size(), first.Features.DenseArray(), len(sample))
    weights := make([]float64, first.Features.Size())
    alpha := 0.01
    l1norm := 0.0
    deltas := make([]float64, first.Features.Size())
    for iter := 0; iter < iterNum; iter++ {
        pnum := 0
        errValue := 0.0
        for _, lp := range sample {
            if lp.Label > 0 {
                pnum++
            }
            dotSum := lp.Features.DotDense(weights)
            hypothesis := 1.0 / (1.0 + math.Exp(-dotSum))
            loss := hypothesis - lp.Label
            errValue += math.Pow(loss, 2) / 2
            for _, i := range lp.Features.Indices() {
                v := lp.Features.Get(i)
                deltas[i] = deltas[i] - alpha * (loss * v + weights[i] * l1norm)
            }
        }
        for i, v := range deltas {
            weights[i] = v
        }
        utils.DebugLog.Write("BGD iter %d: loss %.6f %v", iter, errValue, pnum)
    }
    log.Println(weights)
    m.weights = NewVectorWithDense(weights)
}
func predict(weights, features *Vector) float64 {
    if weights.Size() == features.Size() {
        dotSum := weights.Dot(features)
        return 1.0 / (1.0 + math.Exp(-dotSum))
    }
    return 0
}