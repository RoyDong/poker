package ml

import (
    "strings"
    "strconv"
    "errors"
    "sync"
    "sort"
    "dw/poker/utils"
)

type IsotonicRegression struct {
    *utils.FileLoader
    version     string
    locker      sync.RWMutex
    boundaries  []float64
    predictions []float64
}

func (m *IsotonicRegression) Version() string {
    return m.version
}

func (m *IsotonicRegression) Predict(v float64) float64 {
    m.locker.RLock()
    defer m.locker.RUnlock()
    index := sort.SearchFloat64s(m.boundaries, v)
    if index == 0 {
        return m.predictions[0]
    }
    if index == len(m.boundaries) {
        return m.predictions[len(m.predictions) - 1]
    }
    //v 处于boundary分段之间
    if m.boundaries[index] > v {
        return linearInterpolation(
            m.boundaries[index - 1],
            m.predictions[index - 1],
            m.boundaries[index],
            m.predictions[index],
            v)
    }
    return m.predictions[index]
}

func linearInterpolation(x1, y1, x2, y2, x float64) float64 {
    return y1 + (y2 - y1) * (x - x1) / (x2 - x1)

}

func (m *IsotonicRegression) newContent(bytes []byte) error {
    var err error
    var headBody = strings.Split(string(bytes), "\r\n")
    if len(headBody) != 2 {
        err = errors.New("cpc/ml: model data malformed")
        utils.FatalLog.Write(err.Error())
        return err
    }

    /*
    TODO head
    if rows := strings.Split(headBody[0], "\n"); len(rows) > 0 {
        for _, row := range rows {
            vals := strings.SplitN(strings.TrimSpace(row), " ", 2)
            if len(vals) == 2 {
                k := strings.TrimSpace(vals[0])
                v := strings.TrimSpace(vals[1])
                if len(k) > 0 {
                    switch k {

                    //TODO
                    case "version":
                    case "num_data":
                    case "bin_num":
                    case "date":
                    case "mean_squared_error":
                    case "mean_error":
                    }
                }
            }
        }

    }
    */

    boundaries := make([]float64, 0)
    predictions := make([]float64, 0)
    if rows := strings.Split(headBody[1], "\n"); len(rows) > 0 {
        for _, row := range rows {
            vals := strings.SplitN(strings.TrimSpace(row), " ", 2)
            if len(vals) == 2 {
                k := strings.TrimSpace(vals[0])
                v := strings.TrimSpace(vals[1])
                if len(k) > 0 {
                    b, err := strconv.ParseFloat(k, 64)
                    if err != nil {
                        utils.FatalLog.Write("cpc/ml: bad model data, boundary must be float64")
                        return err
                    }
                    p, err := strconv.ParseFloat(v, 64)
                    if err != nil {
                        utils.FatalLog.Write("cpc/ml: bad model data, prediction must be float64")
                        return err
                    }
                    boundaries = append(boundaries, b)
                    predictions = append(predictions, p)
                }
            }
        }
        m.locker.Lock()
        m.boundaries = boundaries
        m.predictions = predictions
        m.locker.Unlock()
        utils.WarningLog.Write("cpc/ml IR model loaded %s", m.version)
    } else {
        utils.FatalLog.Write("cpc/ml: bad model data [%s]", m.version)
        return errors.New("cpc/ml: bad model data")
    }
    return nil
}
