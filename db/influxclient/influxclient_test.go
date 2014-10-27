package influxclient

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestGet(t *testing.T) {
	// setup
	methods := []string{"diff", "fence", "magnitude"}
	anomalyClient, err := Setup("influx_config.json", 30, 0, 100, 1, methods, "1h", "mean")
	assert.Equal(t, err, nil, "Error generating anomalyzer: ", err)

	_, err = anomalyClient.Get()
	assert.Equal(t, err, nil, "Error querying data from InfluxDB")
}

func TestUpdate(t *testing.T) {
	// setup
	methods := []string{"diff", "fence", "magnitude"}
	anomalyClient, err := Setup("influx_config.json", 30, 0, 100, 1, methods, "1m", "mean")
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
	anomalyClient, err := Setup("influx_config.json", 30, 0, 100, 1, methods, "1m", "mean")
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
	anomalyClient, err = Setup("influx_config.json", 30, 0, 50, 1, methods, "", "")
	assert.Equal(t, err, nil, "Error generating anomalyzer: %v\n", err)

	// get and update data
	err = anomalyClient.GetAndUpdate()
	assert.Equal(t, err, nil, "Error getting and updating data.")

	// push on a very large value
	prob = anomalyClient.Anomalyzer.Push(100.0)
	assert.NotEqual(t, prob, 1.0, "This behavior should be anomalous.")

}
