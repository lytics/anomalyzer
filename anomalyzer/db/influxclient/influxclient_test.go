package influxclient

import (
	"fmt"
	"github.com/bmizerany/assert"
	influx "github.com/influxdb/influxdb/client"
	anomalyzer "github.com/lytics/anomalyzer/anomalyzer"
	"net/http"
	"testing"
	"time"
)

func makeClient() InfluxAnomalyClient {
	defaults := &influx.ClientConfig{
		Host:       "192.168.115.68:8086",
		Username:   "root",
		Password:   "root",
		Database:   "lytics",
		HttpClient: http.DefaultClient,
	}
	client, err := influx.NewClient(defaults)
	if err != nil {
		fmt.Println("Error generating new client")
	}

	conf := &anomalyzer.AnomalyzerConf{
		// this upper bound reflects 30% CPU Usage
		UpperBound:    30,
		LowerBound:    anomalyzer.NA,
		ActiveSize:    20,
		ReferenceSize: 80,
		Methods:       []string{"rank", "fence", "magnitude"},
	}
	anom, _ := anomalyzer.NewAnomalyzer(conf, nil)

	duration, _ := time.ParseDuration("1h")

	anomalyClient := InfluxAnomalyClient{
		Client:      client,
		Anomalyzer:  &anom,
		Table:       "metd.lio5.elasticsearch.cpu.avg",
		Updated:     time.Now(),
		Granularity: duration,
	}
	return anomalyClient
}

func TestGet(t *testing.T) {
	anomalyClient := makeClient()
	_, err := anomalyClient.Get()
	assert.Equal(t, err, nil, "Error querying data from InfluxDB")
}

func TestUpdate(t *testing.T) {
	anomalyClient := makeClient()
	ys, err := anomalyClient.Get()
	assert.Equal(t, err, nil, "Error querying data from InfluxDB")

	// initial anomalyzer.data is empty list,
	// let's populate it
	err = anomalyClient.Update(ys)
	predata := anomalyClient.Anomalyzer.Data
	pretime := anomalyClient.Updated.Unix()
	assert.Tf(t, len(predata) > 0, "Underlying data was not filled in")
	assert.Equal(t, err, nil, "Error updating underlying data")

	// wait 45 seconds (enough time for some new data to come in)
	time.Sleep(45 * time.Second)
	ys, _ = anomalyClient.Get()
	_ = anomalyClient.Update(ys)
	postdata := anomalyClient.Anomalyzer.Data
	posttime := anomalyClient.Updated.Unix()

	assert.Tf(t, predata[0] != postdata[0], "Underlying data was not updated")
	assert.Tf(t, pretime < posttime, "Timestamp was not updated")
}
