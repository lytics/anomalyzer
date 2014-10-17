package anomalyze

import (
	"fmt"
	"github.com/drewlanenga/govector"
	"math"
)

const (
	NA = math.SmallestNonzeroFloat64
)

type AnomalyzerConf struct {
	//Sensitivity   float64
	UpperBound    float64
	LowerBound    float64
	ActiveSize    int
	ReferenceSize int
	PermCount     int
	Methods       []string
}

type Anomalyzer struct {
	Conf *AnomalyzerConf
	Data govector.Vector
}

func validateConf(conf *AnomalyzerConf) error {
	// if supplied, make sure the detection methods are supported
	supportedMethods := []string{"magnitude", "diff", "rank", "fence", "ks"}
	minimumMethods := []string{"magnitude", "diff"}
	if conf.Methods == nil {
		conf.Methods = minimumMethods
	} else {
		for _, method := range conf.Methods {
			if !exists(method, supportedMethods) {
				return fmt.Errorf("Unsupported detection method '%s'", method)
			}
		}
	}

	// reference window must be at least of size 2 (for difference methods)
	if conf.ReferenceSize < 2 {
		return fmt.Errorf("Reference window must be at least of size 2, (%d) given", conf.ReferenceSize)
	}

	// window sizes must be positive ints
	if conf.ActiveSize < 1 || conf.ReferenceSize < 1 {
		return fmt.Errorf("Active and reference window sizes must be at least of size 1")
	}

	/*
		// validation for the magnitude test
		if conf.Sensitivity < 0 || conf.Sensitivity > 1 {
			return fmt.Errorf("Sensitivity must be between 0 and 1, %v given", conf.Sensitivity)
		}
	*/

	// validation for the fence test
	if exists("fence", conf.Methods) {
		if conf.UpperBound == conf.LowerBound {
			return fmt.Errorf("Fence test included with identical bounds on the fences")
		} else {
			if conf.UpperBound < conf.LowerBound {
				return fmt.Errorf("UpperBound (%v) was lower than the LowerBound (%v)", conf.UpperBound, conf.LowerBound)
			}
		}
	}

	// validation for the rank test
	if exists("rank", conf.Methods) || exists("ks", conf.Methods) {
		if conf.PermCount == 0 {
			conf.PermCount = 500
		}
	}

	// validation for diff test
	if exists("diff", conf.Methods) {
		if conf.ReferenceSize < 4 {
			return fmt.Errorf("The Difference method requires a minimum reference window of 4 points, (%v) given", conf.ReferenceSize)
		}
	}

	return nil
}

func index(needle string, haystack []string) int {
	for i, straw := range haystack {
		if straw == needle {
			return i
		}
	}
	return -1

}

func exists(needle string, haystack []string) bool {
	return index(needle, haystack) > -1
}

func NewAnomalyzer(conf *AnomalyzerConf, data []float64) (Anomalyzer, error) {
	err := validateConf(conf)
	if err != nil {
		return Anomalyzer{}, err
	}

	vector, err := govector.AsVector(data)
	if err != nil {
		return Anomalyzer{}, err
	}

	return Anomalyzer{conf, vector}, nil
}

func (a *Anomalyzer) Update(x []float64) {
	for _, val := range x {
		a.Data.Push(val)
	}
}

func (a Anomalyzer) Push(x float64) float64 {
	// add the new point to the data
	a.Data.Push(x)

	// evaluate the anomalous probability
	return a.Eval()
}

// Return the weighted average of four statistical tests
// for anomaly detection and return the probability that
// a behavior is anomalous.
func (a Anomalyzer) Eval() float64 {

	probs := make(govector.Vector, len(a.Conf.Methods))

	for i, method := range a.Conf.Methods {
		algorithm := Algorithms[method]
		probs[i] = cap(algorithm(a.Data, *a.Conf), 0, 1)
	}
	// ignore the error since the length of probs and
	// the weights will always be equal

	weights := a.getWeights(probs)
	weighted, _ := probs.WeightedMean(weights)

	// if all the weights are zero, then our weighted mean
	// function attempts to divide by zero which returns a
	// NaN. we'd like it to return 0.
	if math.IsNaN(weighted) {
		weighted = 0
	}
	return weighted
}

// Use essentially similar weights.  However, if either the magnitude
// or fence methods have high probabilities, upweight them significantly.
func (a Anomalyzer) getWeights(probs govector.Vector) govector.Vector {
	nmethods := len(a.Conf.Methods)
	weights := make(govector.Vector, nmethods)
	i := 0
	for i < nmethods {
		weights[i] = 0.5
		i++
	}

	// If either the magnitude and fence methods don't have any
	// probability to contribute, we don't want to hear about it.
	// If they do, we upweight them substantially.

	dynamicWeights := []string{"magnitude", "fence"}
	for _, dynamicWeight := range dynamicWeights {
		methodIndex := index(dynamicWeight, a.Conf.Methods)
		if methodIndex > -1 {
			if probs[methodIndex] > 0.8 {
				weights[methodIndex] = 1.0
			} else {
				weights[methodIndex] = 0.0
			}
		}
	}

	return weights
}
