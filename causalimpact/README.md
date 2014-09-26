
# Causal Impact

 Implementation of Google's [CausalImpact] (http://google-opensource.blogspot.com/2014/09/causalimpact-new-open-source-package.html) project in Go.

## About & Configuration

 Given a time series and definitions of pre- and post-treatment periods, CausalImpact constructs a Bayesian time-series model. This model is used to predict how a time series would have evolved if treatment had never occurred, then considers the difference between that prediction and the actual data. The [posterior probability] (http://en.wikipedia.org/wiki/Posterior_probability) that treatment has caused a significant change is returned as well as some [other results] (http://google.github.io/CausalImpact/CausalImpact.html#printing-a-summary-table).

 Google released this project as an [R package] (http://github.com/google/CausalImpact.html). We wrote a wrapper in Go using the "os/exec" package to execute the CausalImpact function in an RScript. In order to get started using it, the size of the active ("post-treatment") window needs to be specified. All behavior before that period is considered the reference ("pre-treatment") window. That information is stored in a struct along with the time series data. The function we wrote returns the posterior probability of a causal impact and a boolean corresponding to whether or not the confidence interval on the relative effect of treatment includes zero. A confidence interval which does not include zero means that the treatment likely had a causal effect.

## Example
 ``` go
 package main

import (
	"fmt"
	"github.com/drewlanenga/govector"
	"github.com/lytics/anomalyzer/causalimpact"
)

func main() {
	conf := &causalimpact.CausalImpactConf{
		ActiveSize: 1,
	}

	// initialize with empty data or an actual slice of floats
	// the last entry of data should be large enough to mark a
	// significant increase. change this value to something more
	// reasonable to see how the result of CausalImpact() changes
	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55, 22}
	vector, _ := govector.AsVector(data)
	x := causalimpact.CausalImpactStruct{conf, vector}

	// ("impact.r" needs to be located in the same folder as this code)
	prob, boo, _ := x.CausalImpact()
	if boo == true {
		fmt.Printf("It is likely that a causal effect has occurred. The posterior probability of causation is: %v\n", prob)
	} else {
		fmt.Printf("It is unlikely that a causal effect has occurred.\n")
	}
}
```
