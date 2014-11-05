package influxclient_test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/bmizerany/assert"
	influx "github.com/influxdb/influxdb/client"
	. "github.com/lytics/anomalyzer/db/influxclient"
)

const defaultDB = "testdb"

var (
	testSeries = influx.Series{
		Name:    "test_series",
		Columns: []string{"time", "value"},
		Points:  [][]interface{}{},
	}
	dbInit = sync.Once{}
)

func init() {
	value := 0
	for i := 0; i < 1000; i++ {
		if i == 950 {
			value = 100
		} else {
			value = 10
		}
		testSeries.Points = append(testSeries.Points, []interface{}{i, value})
	}
}

func setupInflux(t *testing.T) *influx.Client {
	conf := &influx.ClientConfig{
		Host:     os.Getenv("INFLUXDB_HOST"),
		Username: os.Getenv("INFLUXDB_USER"),
		Password: os.Getenv("INFLUXDB_PASS"),
		Database: os.Getenv("INFLUXDB_DB"),
	}
	if conf.Database == "" {
		conf.Database = defaultDB
	}
	c, err := influx.NewClient(conf)
	if err != nil {
		t.Fatalf("Error creating influx client: %v", err)
	}
	if err := c.Ping(); err != nil {
		t.Skipf("Skipping InfluxDB tests because it doesn't appear to be running:\n%v", err)
	}
	if _, err = c.Query(fmt.Sprintf("select time, value from %s limit 1", testSeries.Name)); err != nil {
		dbInit.Do(func() {
			t.Log("No test series found. Initializing test db.")
			c.DeleteDatabase(conf.Database)
			c.CreateDatabase(conf.Database)
			if err := c.WriteSeries([]*influx.Series{&testSeries}); err != nil {
				t.Fatalf("Error writing test data: %v", err)
			}
		})
	}
	return c
}

func TestGet(t *testing.T) {
	// setup
	methods := []string{"diff", "fence", "magnitude"}
	ic := setupInflux(t)
	anomalyClient, err := New(ic, testSeries.Name, 0.1, 30, 0, 100, 1, methods, "1h", "mean")
	assert.Equal(t, err, nil, "Error generating anomalyzer: ", err)

	_, err = anomalyClient.Get()
	assert.Equal(t, err, nil, "Error querying data from InfluxDB")
}

func TestUpdate(t *testing.T) {
	// setup
	methods := []string{"diff", "fence", "magnitude"}
	ic := setupInflux(t)
	anomalyClient, err := New(ic, testSeries.Name, 0.1, 30, 0, 100, 1, methods, "1m", "mean")
	assert.Equal(t, err, nil, "Error generating anomalyzer: %v\n", err)

	ys, err := anomalyClient.Get()
	assert.Equal(t, err, nil, "Error querying data from InfluxDB")

	// initial anomalyzer.data is empty list,
	// let's populate it
	err = anomalyClient.Update(ys)
	predata := anomalyClient.Anomalyzer.Data
	assert.Tf(t, len(predata) > 0, "Underlying data was not filled in")
	assert.Equal(t, err, nil, "Error updating underlying data")
}

func TestEval(t *testing.T) {
	// setup
	methods := []string{"diff", "fence", "magnitude"}
	ic := setupInflux(t)
	anomalyClient, err := New(ic, testSeries.Name, 0.1, 30, 0, 100, 1, methods, "1m", "mean")
	assert.Equal(t, err, nil, "Error generating anomalyzer: %v\n", err)

	// get and update data
	err = anomalyClient.GetAndUpdate()
	assert.Equal(t, err, nil, "Error getting and updating data.")

	prob := anomalyClient.Eval()
	assert.Tf(t, prob >= 0.0, "Probability of anomalous behavior is greater than or equal to zero.")
	assert.Tf(t, prob <= 1.0, "Probability of anomalous behavior is less than or equal to one.")

	// setup
	methods = []string{"fence", "magnitude", "diff"}
	// in order to increase sensitivity, no longer averaging values over a relatively large window
	anomalyClient, err = New(ic, testSeries.Name, 0.1, 30, 0, 50, 1, methods, "", "")
	assert.Equal(t, err, nil, "Error generating anomalyzer: %v\n", err)

	// get and update data
	err = anomalyClient.GetAndUpdate()
	assert.Equal(t, err, nil, "Error getting and updating data.")

	// push on a very large value
	anomalyClient.Anomalyzer.Update([]float64{100.0, 100.0, 100.0, 100.0, 100.0, 100.0, 100.0, 100.0})
	prob = anomalyClient.Anomalyzer.Eval()
	assert.Tf(t, prob > 0.5, "This behavior should be anomalous. (%f)", prob)

}
