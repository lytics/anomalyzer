package influxclient

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	influx "github.com/influxdb/influxdb/client"
	"github.com/lytics/anomalyzer"
)

const (
	timeLayout = "2006-01-02 15:04:05.999"
)

type InfluxAnomalyClient struct {
	client      *influx.Client
	Anomalyzer  *anomalyzer.Anomalyzer
	series      string
	granularity string
	updated     time.Time
	function    string //{count | mean | sum}
}

// New creates a new InfluxDB Anomaly detection client.
func New(client *influx.Client, series string, sensitivity, upperbound, lowerbound float64, activesize, nseasons int, methods []string, granularity string, function string) (*InfluxAnomalyClient, error) {
	// build anomalyzer
	anomconf := &anomalyzer.AnomalyzerConf{
		Sensitivity: sensitivity,
		UpperBound:  upperbound,
		LowerBound:  lowerbound,
		ActiveSize:  activesize,
		NSeasons:    nseasons,
		Methods:     methods,
	}
	anom, err := anomalyzer.NewAnomalyzer(anomconf, nil)
	if err != nil {
		return nil, err
	}

	// build influx anomaly client
	anomalyClient := &InfluxAnomalyClient{
		client:      client,
		Anomalyzer:  &anom,
		series:      series,
		granularity: granularity,
		//Updated:     initialtime,
		function: function,
	}

	// validate the client
	if err = anomalyClient.validateAnomalyzer(); err != nil {
		return nil, err
	}
	return anomalyClient, nil
}

func exists(needle string, haystack []string) bool {
	for _, value := range haystack {
		if value == needle {
			return true
		}
	}
	return false
}

func validateDuration(s string) (string, error) {
	pattern_numbers, _ := regexp.Compile("[0-9]")
	pattern_letters, _ := regexp.Compile("[a-z]")

	numbers := pattern_numbers.FindAllString(s, -1)
	letters := pattern_letters.FindAllString(s, -1)

	allNumbers := strings.Join(numbers, "")
	allLetters := strings.Join(letters, "")

	if len(allNumbers) == 0 || len(allLetters) == 0 {
		return "", fmt.Errorf("Invalid duration format")
	}

	if !exists(allLetters, []string{"s", "m", "h", "d", "ms"}) {
		return "", fmt.Errorf("Invalid duration format")
	}

	formatted := fmt.Sprintf("%s%s", allNumbers, allLetters)

	if formatted != s {
		return "", fmt.Errorf("Invalid duration format")
	}

	return formatted, nil
}

func (c *InfluxAnomalyClient) validateAnomalyzer() error {
	if len(c.granularity) != 0 {
		duration, err := validateDuration(c.granularity)
		if err != nil {
			return err
		}
		c.granularity = duration

		if len(c.granularity) != 0 {
			if c.function == "count" {
				return nil
			} else if c.function == "mean" {
				return nil
			} else if c.function == "sum" {
				return nil
			} else {
				err := fmt.Errorf("Granularity was specified, but an aggregate function was not (%s).", c.function)
				return err
			}
		}
	}
	if len(c.granularity) == 0 {
		if c.function == "count" || c.function == "mean" || c.function == "sum" {
			err := fmt.Errorf("An aggregate function was specified, but granularity was not.")
			return err
		}
	}

	return nil
}

// Get data from InfluxDB.
func (c *InfluxAnomalyClient) Get() ([]float64, error) {
	// the number of elements we want to grab
	sampleSize := (1 + c.Anomalyzer.Conf.NSeasons) * c.Anomalyzer.Conf.ActiveSize

	// this query selects the most recent data points over the past day
	// using a "where" avoids scanning the whole set of data
	updated := c.updated.Format(timeLayout)

	var query string
	var index int
	if len(c.granularity) != 0 {
		query = fmt.Sprintf("select %s(value) as value, time from %s where time > '%s' group by time(%s) limit %v",
			c.function, c.series, updated, c.granularity, sampleSize)
		// this query outputs the columns: [time value]
		index = 1
	} else {
		query = fmt.Sprintf("select * from %s where time > '%s' limit %v", c.series, updated, sampleSize)
		// this query outputs the columns : [time sequence_number value]
		index = 2
	}

	series, err := c.client.QueryWithNumbers(query)
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
		val = points[i][index].(json.Number)
		y[j], err = val.Float64()
		if err != nil {
			return nil, err
		}
		j++
		i--
	}
	return y, nil
}

// Update the underlying data in the anomalyzer with the new slice
func (c *InfluxAnomalyClient) Update(data []float64) error {
	// push in new data
	c.Anomalyzer.Update(data)
	c.updated = time.Now()
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

// Eval returns the probability that behavior in the active window is anomalous.
func (c *InfluxAnomalyClient) Eval() float64 {
	return c.Anomalyzer.Eval()
}
