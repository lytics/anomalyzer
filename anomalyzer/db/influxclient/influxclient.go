package influxclient

import (
	"encoding/json"
	"fmt"
	influx "github.com/influxdb/influxdb/client"
	anomalyze "github.com/lytics/anomalyzer/anomalyzer"
	"time"
)

const (
	TIME_LAYOUT = "2006-01-02 15:04:05.999"
)

type InfluxAnomalyClient struct {
	Client      *influx.Client
	Anomalyzer  *anomalyze.Anomalyzer
	Table       string
	Granularity time.Duration
	Updated     time.Time
}

// get data from influx
func (c *InfluxAnomalyClient) Get() ([]float64, error) {
	// the number of elements we want to grab
	sampleSize := c.Anomalyzer.Conf.ActiveSize + c.Anomalyzer.Conf.ReferenceSize
	// this query selects the most recent data points over the past day
	// using a "where" avoids scanning the whole set of data
	updated := c.Updated.Format(TIME_LAYOUT)
	query := fmt.Sprintf("select * from %s where time > '%s' group by time(%s) limit %v", c.Table, updated, c.Granularity.String(), sampleSize)

	series, err := c.Client.QueryWithNumbers(query)
	if err != nil {
		return nil, err
	}
	points := series[0].GetPoints()

	y := make([]float64, len(points))
	var val json.Number

	// y has the most recent entries at the top of its list, we need
	// to reverse the order of this list
	i := len(points) - 1
	j := 0
	for i >= 0 {
		val = points[i][2].(json.Number)
		y[j], err = val.Float64()
		if err != nil {
			return nil, err
		}
		j++
		i--
	}
	return y, nil
}

// update the underlying data in the anomalyzer with the new slice
func (c *InfluxAnomalyClient) Update(data []float64) error {
	// get rid of old data
	var newArray []float64
	c.Anomalyzer.Data = newArray
	// push in new data
	c.Anomalyzer.Update(data)
	c.Updated = time.Now()
	return nil
}

func (c *InfluxAnomalyClient) GetAndUpdate() error {
	data, err := c.Get()
	if err != nil {
		return err
	}

	c.Update(data)
	return nil
}

// return the probability that behavior in the active window is anomalous
func (c *InfluxAnomalyClient) Eval() float64 {
	return c.Anomalyzer.Eval()
}
