package influxclient

import (
	"encoding/json"
	"fmt"
	influx "github.com/influxdb/influxdb/client"
	anomalyzer "github.com/lytics/anomalyzer"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	TIME_LAYOUT = "2006-01-02 15:04:05.999"
)

type Config struct {
	Host     string
	Username string
	Password string
	Database string
	Table    string
}

type InfluxAnomalyClient struct {
	Client      *influx.Client
	Anomalyzer  *anomalyzer.Anomalyzer
	Table       string
	Granularity string
	Updated     time.Time
	Function    string //{count | mean | sum}
}

func Setup(filepath string, upperbound, lowerbound float64, activesize, nseasons int, methods []string, granularity string, function string) (InfluxAnomalyClient, error) {
	// unmarshal db stuff from json file
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return InfluxAnomalyClient{}, err
	}

	var conf Config
	err = json.Unmarshal(content, &conf)
	if err != nil {
		return InfluxAnomalyClient{}, err
	}

	// make influx client
	defaults := &influx.ClientConfig{
		Host:       conf.Host,
		Username:   conf.Username,
		Password:   conf.Password,
		Database:   conf.Database,
		HttpClient: http.DefaultClient,
	}

	client, err := influx.NewClient(defaults)
	if err != nil {
		return InfluxAnomalyClient{}, err
	}

	// build anomalyzer
	anomconf := &anomalyzer.AnomalyzerConf{
		UpperBound: upperbound,
		LowerBound: lowerbound,
		ActiveSize: activesize,
		NSeasons:   nseasons,
		Methods:    methods,
	}
	anom, err := anomalyzer.NewAnomalyzer(anomconf, nil)
	if err != nil {
		return InfluxAnomalyClient{}, err
	}

	// validate duration
	//duration, err := validateDuration(granularity)
	//if err != nil {
	//	return InfluxAnomalyClient{}, err
	//}

	// build influx anomaly client
	anomalyClient := InfluxAnomalyClient{
		Client:      client,
		Anomalyzer:  &anom,
		Table:       conf.Table,
		Granularity: granularity,
		Updated:     time.Now(),
		Function:    function,
	}

	// validate the client
	err = anomalyClient.validateAnomalyzer()
	if err != nil {
		return InfluxAnomalyClient{}, err
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
	if len(c.Granularity) != 0 {
		duration, err := validateDuration(c.Granularity)
		if err != nil {
			return err
		}
		c.Granularity = duration

		if len(c.Granularity) != 0 {
			if c.Function == "count" {
				return nil
			} else if c.Function == "mean" {
				return nil
			} else if c.Function == "sum" {
				return nil
			} else {
				err := fmt.Errorf("Granularity was specified, but an aggregate function was not (%s).", c.Function)
				return err
			}
		}
	}
	if len(c.Granularity) == 0 {
		if c.Function == "count" || c.Function == "mean" || c.Function == "sum" {
			err := fmt.Errorf("An aggregate function was specified, but granularity was not.")
			return err
		}
	}

	return nil
}

// get data from influx
func (c *InfluxAnomalyClient) Get() ([]float64, error) {
	// the number of elements we want to grab
	sampleSize := (1 + c.Anomalyzer.Conf.NSeasons) * c.Anomalyzer.Conf.ActiveSize
	// this query selects the most recent data points over the past day
	// using a "where" avoids scanning the whole set of data
	updated := c.Updated.Format(TIME_LAYOUT)

	var query string
	if len(c.Granularity) != 0 {
		query = fmt.Sprintf("select %s(value) as value, time from %s where time > '%s' group by time(%s) limit %v", c.Function, c.Table, updated, c.Granularity, sampleSize)
	} else {
		query = fmt.Sprintf("select * from %s where time > '%s' limit %v", c.Table, updated, sampleSize)
	}

	series, err := c.Client.QueryWithNumbers(query)
	if err != nil {
		fmt.Printf("Query: %s\n", query)
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
		val = points[i][1].(json.Number)
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
