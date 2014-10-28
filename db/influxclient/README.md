
# Application of Anomalyzer on InfluxDB

Grab timeseries data from InfluxDB using the [Go client library](http://github.com/influxdb/influxdb/tree/master/client) and detect anomalies on it using the [Anomalyzer package](https://github.com/lytics/anomalyzer/tree/master/anomalyzer).

## InfluxDB

InfluxDB is a time series database written in Go. In order to get started accessing a database, the `Host`, `Username`, `Password`, `Database`, and `Table` need to be specified so that a client can be created. This information should be kept in a json file as such:
``` json
{
	"Host":       "ip address",
	"Username":   "username",
	"Password":   "password",
	"Database":   "database_name",
	"Table": 	  "table_name"
}
```
The client created can then be used to query the database using [InfluxDB's query language](http://influxdb.com/docs/v0.7/api/query_language.html). Additionally, upper and lower bounds, active window length, and number of seasons need to be specified in order to run the anomalyzer package. Granularity and an aggregate function are optional arguments, which when both specified add a ["group by"](http://influxdb.com/docs/v0.8/api/query_language.html#group-by) clause to the query.

To grab new data, the `Get` function can be used. It queries the database for the most recent points (to avoid scanning a large set of data). The number of points selected depends on `ActiveSize` and `NSeasons`.

## Anomalyzer

### Example

Consider the case of monitoring CPU usage.  (Specific configuration should be adjusted specifically to the application a user is considering.) 

Let's assume we collect a new point every 30 seconds.  We choose `ActiveSize` to be 120, to consider recent activity in the past hour, and `NSeasons` to be 1, to allow us to compare activity this hour to activity last hour. We also choose an `UpperBound` of 30, which allows us to make sure that we are alerted as CPU usage approaches and exceeds 30%.  (Setting the lower bound to `NA` lets the anomalyzer know that we don't care if usage stays low.)

The [algorithms](https://github.com/lytics/anomalyzer/tree/master/anomalyzer#algorithms) applied were **diff**, **fence**, and **magnitude**. We chose not to apply **cdf** because it can be sensitive to small fluctuations, which be frequent in usage data.

After grabbing new data using `Get`, the underlying data can be updated using `Update`. The `Eval` function returns a probability that behavior in the active window is anomalous by accessing the Anomalyzer package.

## Example
``` go
package main

import (
	"fmt"
	influx "github.com/influxdb/influxdb/client"
	"github.com/lytics/anomalyzer"
	influxclient "github.com/lytics/anomalyzer/db"
)


func main() {
	// specify: path to config.json file, upper bound, lower bound,
	// length of the active window, number of seasons, list of methods,
	// granularity and an aggregate function
	methods := []string{"diff", "fence", "magnitude"}
	anomalyClient, err := Setup("config.json", 30, anomalyzer.NA, 100, 1, methods, "1h", "mean")

	// query existing database to get set
	ys, _ := anomalyClient.Get()
	// update client with this new data
	_ = anomalyClient.Update(ys)

	// or use GetAndUpdate() fn on anomalyClient

	// now you can run the anomalize package over this
	// set like this
	prob := anomalyClient.Eval()
	fmt.Printf("Anomalous Probability: %v\n", prob)
}

```
