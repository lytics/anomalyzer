
# Application of Anomalyzer on InfluxDB

Grab timeseries data from InfluxDB using the [Go client library](http://github.com/influxdb/influxdb/tree/master/client) and detect anomalies on it using the [Anomalyzer package](https://github.com/lytics/anomalyzer/tree/master/anomalyzer).

## InfluxDB

InfluxDB is a time series database written in Go. In order to get started accessing a database, the `Host`, `Username`, `Password`, and `Database` need to be specified so that a client can be created. This information should be kept in a json file as such:
``` json
{
	"Host":       "ip address",
	"Username":   "username",
	"Password":   "password",
	"Database":   "database_name"
}
```
The client created can then be used to query the database using [InfluxDB's query language](http://influxdb.com/docs/v0.7/api/query_language.html).

To grab new data, the `Get` function can be used. It queries the database for the most recent points within a day (to avoid scanning a large set of data). The number of points selected depends on `ActiveSize` and `NSeasons`.

## Anomalyzer

The type of data we analyzed was CPU usage, which informed our anomalyzer configuration choices. (The following data may need to be tweaked for whatever application a user is considering.) 

We chose `ActiveSize` to be 100 and `NSeasons` to be 1, which for us corresponded to reference and active windows a little under an hour. An `UpperBound` of 30 was chosen, which in this application referred to a percentage of CPU usage. The [algorithms](https://github.com/lytics/anomalyzer/tree/master/anomalyzer#algorithms) applied were **diff**, **fence**, and **magnitude**. We chose not to apply **cdf** because it was very sensitive to small fluctuations which there were a lot of in this CPU usage data.

After grabbing new data using `Get`, the underlying data can be updated using `Update`. The `Eval` function returns a probability that behavior in the active window is anomalous by accessing the Anomalyzer package.

## Example
``` go
package main

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

func main() {
	// include the filepath to the json with influx information
	defaults := Setup("config.json")
	client, err := influx.NewClient(defaults)
	if err != nil {
		fmt.Println("Error generating new client")
	}

	// anomalyzer set-up
	conf := &anomalyzer.AnomalyzerConf{
		UpperBound:    30,
		LowerBound:    anomalyzer.NA,
		ActiveSize:    100,
		NSeasons:      1,
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
		Table:   "series",
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
