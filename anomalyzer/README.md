
# Anomalyzer

Probability-based anomaly detection in Go.

## Windows

Anomalyzer implements a suite of statistical tests that yield the probability that a given set of numeric input, typically a time series, contains anomalous behavior.  Each test compares the behavior in an **active window** or one or more points to the behavior in a **reference window** of two or more points.

For example, an input vector of `[1, 2, 3, 4, 5, 6, 7, 8, 9]`, and an active window length of 1 and a reference window length of 4, would yield an active window of `[9]` and a reference window of `[5, 6, 7, 8]`.

## Algorithms

Anomalyzer can implement one or more of the following algorithmic tests:

1. **diff**: Compares the differences in the behavior in the active window to the cumulative distribution function of the reference window.
2. **rank**: Performs a bootstrap permutation test on the ranks of the differences in both windows, in the flavor of a [Mann-Whitney](http://en.wikipedia.org/wiki/Mann%E2%80%93Whitney_U_test) test.
3. **magnitude**: Compares the relative magnitude of the difference between the averages of the active window and the reference window.
4. **fence**: Indicates that data are approaching a configurable upper and lower bound.
5. **bootstrap ks**: Calculates the [Kolmogorov-Smirnov](http://en.wikipedia.org/wiki/Kolmogorov%E2%80%93Smirnov_test) test over active and reference windows and compares that value to KS test scores obtained after permuting all elements in the set

Each test yields a probability of anomalous behavior, and the probabilities are then computed over a weighted mean to determine if the overall behavior is anomalous.  Since a *probability* is returned, the user may determine the sensitivity of the decision, and can determine the threshold for anomalous behavior for the application, whether at say 0.8 for general anomalous behavior or 0.95 for extreme anomalous behavior.

## Configuration

Any of the tests can be included in the anomalyzer, and if none are supplied in the configuration, default to magnitude and diff.  Methods are supplied through the `Methods` value in the configuration and accepts a slice of strings for the method names.

The values for `ActiveSize` and `ReferenceSize` are also required and must be a minimum of 1 and 2, respectively.

After considering reference windows of different lengths, it appears that the **rank** test is slightly more sensitive over a longer reference window, **magnitude** over a shorter reference window, and **diff** is generally applicable when both shorter and larger reference windows are considered. 

### Bootstrap KS

To capture seasonality, **bootstrap ks** should consider an active window equal to a season/period and a reference window equal to more than one season/period. 

### Fence

The fence test can be configured to use custom `UpperBound` and `LowerBound` values for the fences.  If no lower bound is desired, set the value of `LowerBound` to `anomalizer.NA`.

### Rank

The rank test can accept a value for the number of bootstrap samples to generate, indicated by `PermCount`, and defaults to 500 if not set.


## Example

```go
package main

import (
	"fmt"
	anomalyze "github.com/lytics/anomalyzer/anomalyzer"
)

func main() {
	conf := &anomalyze.AnomalyzerConf{
		UpperBound:    5,
		LowerBound:    0,
		ActiveSize:    1,
		ReferenceSize: 4,
		Methods:       []string{"diff", "fence", "rank", "magnitude"},
	}

	// initialize with empty data or an actual slice of floats
	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55}

	anomalyzer, _ := anomalyze.NewAnomalyzer(conf, data)

	// the push method automatically triggers a recalcuation of the
	// anomaly probability.  The recalculation can also be triggered
	// by a call to the Eval method.
	prob := anomalyzer.Push(8.0)

	fmt.Println("Anomalous Probability:", prob)
}

```
