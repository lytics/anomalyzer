
# Application of Anomalyzer on InfluxDB

Grab timeseries data from InfluxDB using the [Go client library](http://github.com/influxdb/influxdb/tree/master/client) and detect anomalies on it using the [Anomalyzer package](https://github.com/lytics/anomalyzer/tree/master/anomalyzer).

## InfluxDB

InfluxDB is a time series database written in Go. In order to get started accessing a database, the `Host`, `Username`, `Password`, and `Database` need to be specified in an `influx.ClientConfig` so that a client can be created.

The client created can then be used to query a specific `Table` using [InfluxDB's query language](http://influxdb.com/docs/v0.7/api/query_language.html). Additionally, `Sensitivity`, `UpperBound`, `LowerBound`, `ActiveSize`, and `NSeasons` need to be specified in order to run the anomalyzer package. Granularity and an aggregate function are optional arguments, which when both specified add a ["group by"](http://influxdb.com/docs/v0.8/api/query_language.html#group-by) clause to the query.

To grab new data, the `Get` function can be used. It queries the database for the most recent points (to avoid scanning a large set of data). The number of points selected depends on `ActiveSize` and `NSeasons`. 
After grabbing new data, the underlying data can be updated using `Update`. The `Eval` function returns a probability that behavior in the active window is anomalous by accessing the Anomalyzer package.

### Example

Consider the case of monitoring CPU usage. Let's assume we collect a new point every 30 seconds.  We choose `ActiveSize` to be 2 and `NSeasons` to be 59, to allow us to compare activity in the past minute to the past hour. We also choose an `UpperBound` of 80, which allows us to make sure that we are alerted as CPU usage approaches and exceeds 80%. (Setting the lower bound to `NA` lets the anomalyzer know that we don't care if usage stays low.) The [algorithms](https://github.com/lytics/anomalyzer/tree/master/anomalyzer#algorithms) applied were **ks** and **high rank**. (See EXAMPLES.md file in anomalyzer repository for more detailed analysis.)

``` go
package main

import (
    "fmt"
    influx "github.com/influxdb/influxdb/client"
    "github.com/lytics/anomalyzer"
    influxclient "github.com/lytics/anomalyzer/db/influxclient"
    "log"
)

func main() {
    // specify: influxdb information, sensitivity, upper bound,
    // lower bound, length of the active window, number of seasons,
    // and list of methods. optional: granularity and an aggregate 
    // function (if either are nil, specify "").

    conf := &influx.ClientConfig{
        Host:     "hostname:8086",
        Username: "username",
        Password: "password",
        Database: "database_name",
    }
    client, _ := influx.NewClient(conf)
    methods := []string{"ks", "highrank"}

    anomalyClient, err := influxclient.New(client, "table_name", 0.1, 80.0, anomalyzer.NA, 2, 59, methods, "", "")
    if err != nil {
        log.Fatalf("Error initializing anomalyzer: %v\n", err)
    }

    // query existing database to get set
    ys, err := anomalyClient.Get()
    fmt.Printf("Series: %v\n", ys)
    if err != nil {
        log.Fatalf("Get() Error: %v\n", err)
    }
    // update client with this new data
    err = anomalyClient.Update(ys)
    if err != nil {
        log.Fatalf("Update() Error: %v\n", err)
    }
    // or use GetAndUpdate() fn on anomalyClient

    // now you can run the anomalize package over this
    // set like this
    prob := anomalyClient.Eval()
    fmt.Printf("Anomalous Probability: %v\n", prob)
}
```
