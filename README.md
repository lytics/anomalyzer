# Anomalyzer

[![Build Status](https://travis-ci.org/lytics/anomalyzer.svg?branch=master)](https://travis-ci.org/lytics/anomalyzer) [![GoDoc](https://godoc.org/github.com/lytics/anomalyzer?status.svg)](https://godoc.org/github.com/lytics/anomalyzer)

Probabilistic anomaly detection in Go.

Featured on [Hacker News](https://news.ycombinator.com/item?id=10055977) August 13th, 2015 and the [Lytics Developer Blog](https://www.getlytics.com/developers/blog/post/check_out_anomalyzer).

## Windows

Inspired by Etsy's [Skyline](https://github.com/etsy/skyline) package, Anomalyzer implements a suite of statistical tests that yield the probability that a given set of numeric input, typically a time series, contains anomalous behavior.  Each test compares the behavior in an **active window** of one or more points to the behavior in a **reference window** of two or more points.

Specifying a number of seasons will yield a reference window length equal to that factor times the length of the active window specified. For example, an input vector of `[1, 2, 3, 4, 5, 6, 7, 8, 9]`, and an active window length of 1 with number of seasons equal to 4, would yield an active window of `[9]` and a reference window of `[5, 6, 7, 8]`.

## Algorithms

Anomalyzer can implement one or more of the following algorithmic tests:

1. **cdf**: Compares the differences in the behavior in the active window to the cumulative distribution function of the reference window. 
2. **diff**: Performs a bootstrap permutation test on the ranks of the differences in both windows, in the flavor of a [Wilcoxon rank-sum](http://en.wikipedia.org/wiki/Mann%E2%80%93Whitney_U_test) test.
3. **high rank**: Performs a bootstrap permutation test on the ranks of the entries themselves in both windows. Counts how many times the permuted rank-sum is less than the original rank-sum, sensitive to increasing behavior.
4. **low rank**: Similarly, counts how many times the permuted rank-sum is greater than the original rank-sum, sensitive to decreasing behavior.
5. **magnitude**: Compares the relative magnitude of the difference between the averages of the active and the reference windows.
6. **fence**: Indicates that data are approaching a configurable upper and lower bound.
7. **bootstrap ks**: Calculates the [Kolmogorov-Smirnov](http://en.wikipedia.org/wiki/Kolmogorov%E2%80%93Smirnov_test) test over active and reference windows and compares that value to KS test scores obtained after permuting all elements in the set. 

Each test yields a probability of anomalous behavior, and the probabilities are then computed over a weighted mean to determine if the overall behavior is anomalous.  Since a *probability* is returned, the user may determine the sensitivity of the decision, and can determine the threshold for anomalous behavior for the application, whether at say 0.8 for general anomalous behavior or 0.95 for extreme anomalous behavior.

## Configuration

Any of the tests can be included in the anomalyzer, and if none are supplied in the configuration, default to magnitude and cdf.  Methods are supplied through the `Methods` value in the configuration and accepts a slice of strings for the method names.

A value for `ActiveSize`is required and must be a minimum of 1. The `NSeasons` will default to 4 if not specified. 

### Magnitude

If the magnitude test is specified, a `Sensitivity` (between 0 and 1) can be supplied such that when the result of the magnitude test is less than that value, the weighted mean will return 0. If `Sensitivity` is not specified, it defaults to 0.1.

### Bootstrap KS

To capture seasonality, the bootstrap ks test should consider an active window length equal to a season. 

### Fence

The fence test can be configured to use custom `UpperBound` and `LowerBound` values for the fences.  If no lower bound is desired, set the value of `LowerBound` to `anomalyzer.NA`.

### Diff & Rank

The diff, bootstrap ks, and rank tests can accept a value for the number of bootstrap samples to generate, indicated by `PermCount`, and defaults to 500 if not set.


## Example

```go
package main

import (
	"fmt"
	"github.com/lytics/anomalyzer"
)

func main() {
	conf := &anomalyzer.AnomalyzerConf{
		Sensitivity: 0.1,
		UpperBound:  5,
		LowerBound:  anomalyzer.NA, // ignore the lower bound
		ActiveSize:  1,
		NSeasons:    4,
		Methods:     []string{"diff", "fence", "highrank", "lowrank", "magnitude"},
	}

	// initialize with empty data or an actual slice of floats
	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55}

	anom, _ := anomalyzer.NewAnomalyzer(conf, data)

	// the push method automatically triggers a recalcuation of the
	// anomaly probability.  The recalculation can also be triggered
	// by a call to the Eval method.
	prob := anom.Push(8.0)
	fmt.Println("Anomalous Probability:", prob)
}
```

