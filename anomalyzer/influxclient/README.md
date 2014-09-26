
# Application of Anomalyzer on InfluxDB

Grab timeseries data from InfluxDB using the [Go client library] (http://github.com/influxdb/influxdb/tree/master/client) and detect anomalies on it using the [Anomalyzer package] (http://github.com/lytics/anomalyzer).

## InfluxDB

InfluxDB is a time series database written in Go. In order to get started accessing a database, the location, username, password, and name associated with that database needs to be specified so that a client can be created. That client can then be used to query the database using [InfluxDB's query language] (http://influxdb.com/docs/v0.7/api/query_language.html).

To grab new data, the Get function can be used. It queries the database for the most recent points within a day (to avoid scanning a large set of data). The number of points selected depends on the sizes of the [reference and active windows] (https://github.com/lytics/anomalyzer#windows) considered.

## Anomalyzer

An anomalyzer instance contains information about window size, an upper bound, types of statistical methods to apply, etc. as well as the underlying data. The type of data we analyzed was CPU usage, which informed our configuration choices. (The following data may need to be tweaked for whatever application a user is considering.) We chose a reference window size of 150 and an active window size of 100, which corresponded to a little over and a little under an hour respectively. An upper bound of 30 was chosen, which in this application refers to a percentage of CPU usage. The [algorithms] (https://github.com/lytics/anomalyzer#algorithms) we chose to apply were "rank", "fence", and "magnitude". We chose not to apply "diff" because it was very sensitive to small fluctuations which there were a lot of in this CPU usage data.

After grabbing new data using Get, the underlying data can be updated using Update. It replaces all previous data with newly aquired data using the Update function within the Anomalyzer package. The Eval function returns a probability that behavior in the active window is anomalous. It also calls on the already implemented Eval function within the Anomalyzer package.

## Example
``` go
package main

import (
	"fmt"
	influx "github.com/influxdb/influxdb/client"
	anomalyzer "github.com/lytics/anomalyzer/anomalyzer"
	"github.com/lytics/anomalyzer/anomalyzer/influxclient"
	"net/http"
	"time"
)

func main() {
	// InfluxDB set-up
	defaults := &influx.ClientConfig{
		Host:       "localhost:8086", // the location of your InfluxDB
		Username:   "root",           //
		Password:   "root",
		Database:   "SampleCPU", // the name of the database you want
		HttpClient: http.DefaultClient,
	}
	client, err := influx.NewClient(defaults)
	if err != nil {
		fmt.Println("Error generating new client")
	}

	// Anomalyzer set-up
	conf := &anomalyzer.AnomalyzerConf{
		UpperBound:    30,
		LowerBound:    anomalyzer.NA,
		ActiveSize:    100,
		ReferenceSize: 150,
		Methods:       []string{"rank", "fence", "magnitude"},
	}
	anom, _ := anomalyzer.NewAnomalyzer(conf, nil)

	// interact with an InfluxAnomalyClient which contains
	// the InfluxDB client information and the Anomalyzer
	// set-up
	anomalyClient := influxclient.InfluxAnomalyClient{
		Client:     client,
		Anomalyzer: &anom,
		// name of the series we are interested in
		Table:   "metd.lio5.elasticsearch.cpu.avg",
		Updated: time.Now(),
	}

	// query existing database to get set
	ys, _ := anomalyClient.Get()
	// update client with this new data
	_ = anomalyClient.Update(ys)

	// now you can run the anomalize package over this
	// set like this
	prob := anomalyClient.Eval()
	fmt.Printf("Anomalous Probability: %v\n", prob)
}

```