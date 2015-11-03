package anomalyzer

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/drewlanenga/govector"
)

// Generates a random walk given the number of steps desired, a starting point,
// and the desired standard deviation.
func randomWalk(nsteps int, start float64, sd float64) (govector.Vector, error) {
	walk := make(govector.Vector, nsteps)
	walk[0] = float64(start)

	i := 1
	for i < nsteps {
		step := rand.NormFloat64() * sd
		walk[i] = cap(walk[i-1]+step, 0.0, 1.0)

		i++
	}
	return walk, nil
}

func TestConfSetup(t *testing.T) {
	conf := &AnomalyzerConf{
		Sensitivity: 0.1,
		UpperBound:  5,
		LowerBound:  0,
		ActiveSize:  1,
		NSeasons:    4,
		Methods:     []string{"cdf", "fence", "highrank", "lowrank", "magnitude"},
	}
	anomalyzer, err := NewAnomalyzer(conf, []float64{})
	assert.Equal(t, nil, err, "Error initializing new anomalyzer")
	if anomalyzer.Conf.VectorCap < 10000000 {
		//VectorCap should be the maximum integer
		t.Errorf("VectorCap set incorrectly: %#v", anomalyzer.Conf.VectorCap)
	}
}

func TestAnomalyzer(t *testing.T) {
	conf := &AnomalyzerConf{
		Sensitivity: 0.1,
		UpperBound:  5,
		LowerBound:  0,
		ActiveSize:  1,
		NSeasons:    4,
		Methods:     []string{"cdf", "fence", "highrank", "lowrank", "magnitude"},
	}

	// initialize with empty data or an actual slice of floats
	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55}

	anomalyzer, err := NewAnomalyzer(conf, data)
	assert.Equal(t, nil, err, "Error initializing new anomalyzer")

	prob := anomalyzer.Push(8.0)
	assert.Tf(t, prob > 0.5, "Anomalyzer returned a probability that was too small")
}

func TestAnomalyzerCapped(t *testing.T) {
	conf := &AnomalyzerConf{
		Sensitivity: 0.1,
		UpperBound:  5,
		LowerBound:  0,
		ActiveSize:  1,
		NSeasons:    4,
		Methods:     []string{"cdf", "fence", "highrank", "lowrank", "magnitude"},
		VectorCap:   6,
	}

	// initialize with empty data or an actual slice of floats
	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55}

	anomalyzer, err := NewAnomalyzer(conf, data)
	assert.Equal(t, nil, err, "Error initializing new anomalyzer")

	prob, err := anomalyzer.PushCapped(8.0)
	prob, err = anomalyzer.PushCapped(8.0)
	prob, err = anomalyzer.PushCapped(8.0)
	assert.Tf(t, prob > 0.5, "Anomalyzer returned a probability that was too small")
	assert.Equal(t, len(anomalyzer.Data), anomalyzer.Conf.VectorCap)
}

func TestAnomalyzerPushFixed(t *testing.T) {
	conf := &AnomalyzerConf{
		Sensitivity: 0.1,
		UpperBound:  5,
		LowerBound:  0,
		ActiveSize:  1,
		NSeasons:    4,
		Methods:     []string{"cdf", "fence", "highrank", "lowrank", "magnitude"},
	}

	// initialize with empty data or an actual slice of floats
	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55}

	anomalyzer, err := NewAnomalyzer(conf, data)
	assert.Equal(t, nil, err, "Error initializing new anomalyzer")

	prob, err := anomalyzer.PushFixed(8.0)
	prob, err = anomalyzer.PushFixed(10.0)
	prob, err = anomalyzer.PushFixed(8.0)
	prob, err = anomalyzer.PushFixed(9.0)
	assert.Equal(t, err, nil, "There was an error with array size")
	assert.Tf(t, prob > 0.5, "Anomalyzer returned a probability that was too small")
	assert.Equal(t, len(anomalyzer.Data), 6, "Array size did not stay at original size")
}

func TestAnomalyzerPushMixed(t *testing.T) {
	conf := &AnomalyzerConf{
		Sensitivity: 0.1,
		UpperBound:  5,
		LowerBound:  0,
		ActiveSize:  1,
		NSeasons:    4,
		Methods:     []string{"cdf", "fence", "highrank", "lowrank", "magnitude"},
		VectorCap:   8,
	}

	// initialize with empty data or an actual slice of floats
	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55}

	anomalyzer, err := NewAnomalyzer(conf, data)
	assert.Equal(t, nil, err, "Error initializing new anomalyzer")

	prob, err := anomalyzer.PushFixed(8.5)
	prob = anomalyzer.Push(10.0)
	prob, err = anomalyzer.PushFixed(8.0)
	prob = anomalyzer.Push(9.0)
	prob, err = anomalyzer.PushCapped(9.0)
	prob, err = anomalyzer.PushCapped(19.0)
	assert.Equal(t, err, nil, "There was an error with mixing array extension")
	assert.Tf(t, prob > 0.5, "Anomalyzer returned a probability that was too small")
	assert.Equal(t, len(anomalyzer.Data), 8, "Array size Push* functions failed to grow Data to expected size")
	assert.Equal(t, anomalyzer.Data[7], 19.0)
	assert.Equal(t, anomalyzer.Data[0], 2.6, "Two values were appended, two values were popped from the array. 3rd original element should be tail.")
}

func Example() {
	conf := &AnomalyzerConf{
		Sensitivity: 0.1,
		UpperBound:  5,
		LowerBound:  NA, // ignore the lower bound
		ActiveSize:  1,
		NSeasons:    4,
		Methods:     []string{"diff", "fence", "highrank", "lowrank", "magnitude"},
	}

	// initialize with empty data or an actual slice of floats
	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55}

	anom, _ := NewAnomalyzer(conf, data)

	// the push method automatically triggers a recalcuation of the
	// anomaly probability.  The recalculation can also be triggered
	// by a call to the Eval method.
	prob := anom.Push(8.0)
	fmt.Println("Anomalous Probability:", prob)
}
