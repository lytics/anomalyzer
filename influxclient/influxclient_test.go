package influxclient

import (
	"encoding/json"
	"fmt"
	"github.com/bmizerany/assert"
	influx "github.com/influxdb/influxdb/client"
	anomalyzer "github.com/lytics/anomalyzer/anomalyzer"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

type Config struct {
	Host     string
	Username string
	Password string
	Database string
}

func Setup(filepath string) *influx.ClientConfig {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Print("Error:", err)
	}

	var conf Config
	err = json.Unmarshal(content, &conf)
	if err != nil {
		fmt.Print("Error:", err)
	}

	defaults := &influx.ClientConfig{
		Host:       conf.Host,
		Username:   conf.Username,
		Password:   conf.Password,
		Database:   conf.Database,
		HttpClient: http.DefaultClient,
	}
	return defaults
}

func makeClient() InfluxAnomalyClient {
	defaults := Setup("influx_config.json")

	client, err := influx.NewClient(defaults)
	if err != nil {
		fmt.Println("Error generating new client")
	}

	conf := &anomalyzer.AnomalyzerConf{
		// this upper bound reflects 30% CPU Usage
		UpperBound: 30,
		LowerBound: anomalyzer.NA,
		ActiveSize: 20,
		NSeasons:   4,
		Methods:    []string{"rank", "fence", "magnitude"},
	}
	anom, _ := anomalyzer.NewAnomalyzer(conf, nil)

	anomalyClient := InfluxAnomalyClient{
		Client:     client,
		Anomalyzer: &anom,
		Table:      "metd.lio5.elasticsearch.cpu.avg",
		Updated:    time.Now(),
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
	time.Sleep(60 * time.Second)
	ys, _ = anomalyClient.Get()
	_ = anomalyClient.Update(ys)
	//postdata := anomalyClient.Anomalyzer.Data
	posttime := anomalyClient.Updated.Unix()

	//assert.Tf(t, predata[0] != postdata[0], "Underlying data was not updated")
	assert.Tf(t, pretime < posttime, "Timestamp was not updated")
}
