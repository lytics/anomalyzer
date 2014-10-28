package anomalyzer

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
	referenceSize int
	NSeasons      int
	PermCount     int
	Methods       []string
}

type Anomalyzer struct {
	Conf *AnomalyzerConf
	Data govector.Vector
}

func validateConf(conf *AnomalyzerConf) error {
	// if supplied, make sure the detection methods are supported
	supportedMethods := []string{"magnitude", "diff", "rank", "fence", "ks", "cdf"}
	minimumMethods := []string{"magnitude", "ks"}
	if conf.Methods == nil {
		conf.Methods = minimumMethods
	} else {
		for _, method := range conf.Methods {
			if !exists(method, supportedMethods) {
				return fmt.Errorf("Unsupported detection method '%s'", method)
			}
		}
	}

	// if number of seasons are not specified, default it to 4
	if conf.NSeasons == 0 {
		conf.NSeasons = 4
	}

	// make reference window some multiple of the active window size
	conf.referenceSize = conf.NSeasons * conf.ActiveSize

	// window sizes must be positive ints
	if conf.ActiveSize < 1 {
		return fmt.Errorf("Active window size must be at least of size 1")
	}

	if conf.referenceSize < 4 {
		return fmt.Errorf("The combination of active window (%d) and nseasons (%d) yields a reference window that is too small for analysis.  Please increase one or both.", conf.ActiveSize, conf.NSeasons)
	}

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

	// validation for the permutation tests
	if exists("rank", conf.Methods) || exists("ks", conf.Methods) || exists("diff", conf.Methods) {
		if conf.PermCount == 0 {
			conf.PermCount = 500
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

// Return the weighted average of all statistical tests
// for anomaly detection, which yields the probability that
// the currently observed behavior is anomalous.
func (a Anomalyzer) Eval() float64 {

	probs := make(govector.Vector, 0, len(a.Conf.Methods))
	weights := make(govector.Vector, 0, len(a.Conf.Methods))

	for _, method := range a.Conf.Methods {
		algorithm := Algorithms[method]
		prob := cap(algorithm(a.Data, *a.Conf), 0, 1)

		if prob != NA {
			probs = append(probs, prob)
			weights = append(weights, a.getWeight(method, prob))
		}
	}

	// ignore the error since we force the length of probs
	// and the weights to be equal
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
func (a Anomalyzer) getWeight(name string, prob float64) float64 {
	weight := 0.5

	dynamicWeights := []string{"magnitude", "fence"}
	// If either the magnitude and fence methods don't have any
	// probability to contribute, we don't want to hear about it.
	// If they do, we upweight them substantially.
	if exists(name, dynamicWeights) {
		if prob > 0.8 {
			weight = 1.0
		} else {
			weight = 0.0
		}
	}

	return weight
}
