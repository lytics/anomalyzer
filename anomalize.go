package anomalize

import (
	"fmt"
	"github.com/drewlanenga/govector"
	"math"
	"math/rand"
	"sort"
)

var (
	// These upper and lower bounds of your data and the length of the
	// active window are set universally. These are suggested values.
	lower   = 0.0
	upper   = 1.0
	actsize = 7
)

type Algorithm func(govector.Vector) (float64, error)

// A function will run four statistical tests for anomaly detection
// and return the probability that a behavior is anomalous under each
// test as well as a weighted average of all four tests.
func AnomalyScores(series govector.Vector) []float64 {
	algorithms := map[string]Algorithm{
		"Prob":    Prob,
		"Rank":    Rank,
		"DiffCDF": DiffCDF,
		"Bounds":  Bounds,
	}
	probs := make(govector.Vector, len(algorithms)+1)
	i := 0
	for _, alg := range algorithms {
		prob, err := alg(series)
		if err != nil {

		}
		probs[i], _ = Cap(prob, 0.0, 1.0)
		i++
	}

	// The below weighting scheme is random. Depending on your data type
	// the results of one test could be weighted heavier (see README
	// for description of tests).
	weights := make(govector.Vector, len(algorithms)+1)

	// The result of the bounds test should be selectively weighted, since
	// behavior that is unusual but not close to either the upper or lower
	// limits will have a low result from that test and can bring down your
	// weighted average.
	if probs[3] > 0.8 {
		weights[0] = 0.25
		weights[1] = 0.25
		weights[2] = 0.25
		weights[3] = 0.25
	} else {
		weights[0] = 0.33
		weights[1] = 0.33
		weights[2] = 0.33
		weights[3] = 0.0
	}

	probs[4], _ = probs.WeightedMean(weights)
	return probs
}

// Generates a random walk given the number of steps desired, a starting point,
// and the desired standard deviation.
func RandomWalk(nsteps int, start float64, sd float64) (govector.Vector, error) {
	walk := make(govector.Vector, nsteps)
	walk[0] = float64(start)

	i := 1
	for i < nsteps {
		step := rand.NormFloat64() * sd
		walk[i], _ = Cap(walk[i-1]+step, 0.0, 1.0)

		i++
	}
	return walk, nil
}

func PtoProb(p float64) (float64, error) {
	return (1 - p), nil
}

// Returns a value within a given window (xmin and xmax).
func Cap(x, xmin, xmax float64) (float64, error) {
	if xmin > xmax {
		return 0, fmt.Errorf("Xmin must be greater than or equal to Xmax")
	}
	return math.Max(math.Min(x, xmax), xmin), nil
}

// This function can be used to test whether or not data is getting close to a
// specified upper or lower bound.
func Bounds(vector govector.Vector) (float64, error) {
	totsize := vector.Len()
	active := vector[(totsize - actsize):totsize]

	bound := (upper - lower) / 2
	mid := lower + bound

	x, _ := active.Mean()
	return weight((math.Abs(x-mid))/bound, 10), nil
}

// This is a function will sharply scale values between 0 and 1 such that
// smaller values are weighted more towards 0. A larger base value means a
// more horshoe type function.
func weight(x, base float64) float64 {
	return ((math.Pow(x, base) - 1) / (math.Pow(1, base) - 1))
}

// Generates permutations of reference and active window values to determine
// whether or not data is anomalous. The number of permutations desired has
// been set to 500 but can be increased for more precision.
func Rank(vector govector.Vector) (float64, error) {
	nsteps := 500
	totsize := vector.Len()

	// Find the differences between neighboring elements and rank those differences.
	samp, _ := vector.Diff()
	samp = samp.Apply(math.Abs)
	samp, _ = samp.Rank()

	// Consider the sum of the ranks across the active data. This is the sum that
	// we will compare our permutations to. (The indexing runs to length-1 because
	// after applying .Diff(), we have decreased the length of out vector by 1.)
	summ := samp[(totsize - actsize):(totsize - 1)].Sum()
	i := 0
	count := 0
	tempsumm := 0.0

	// Permute the active and reference data and compute the sums across the tail
	// (from the length of the reference data to the full length).
	for i < nsteps {
		temp, _ := vector.Sample(totsize)
		temp, _ = temp.Diff()
		temp = temp.Apply(math.Abs)
		temp, _ = temp.Rank()

		tempsumm = temp[(totsize - actsize):(totsize - 1)].Sum()

		// If we find a sum that is less than the initial sum across the active data,
		// this implies our initial sum might be uncharacteristically high. We increment
		// our count.
		if tempsumm < summ {
			count++
		}
		i++
	}
	// We return the percentage of the number of iterations where we found our initial
	// sum to be high.
	return float64(count) / float64(nsteps), nil
}

// Generates the cumulative distribution function using the difference in the means
// for the data.
func DiffCDF(vector govector.Vector) (float64, error) {
	totsize := vector.Len()
	reference := vector[0:(totsize - actsize)]
	active := vector[(totsize - actsize):totsize]

	m := reference.Len()

	if m < 2 || actsize < 2 {
		return 0, fmt.Errorf("Length of either vector much be greater than or equal to 2")
	}

	// Find the empircal distribution function using the reference window.
	referencediff, _ := reference.Diff()
	referenceecdffn, _ := referencediff.Ecdf()

	// Difference between the active and reference means.
	activemean, _ := active.Mean()
	referencemean, _ := reference.Mean()
	activediff := activemean - referencemean

	// Apply the empirical distribution function to that difference.
	activeecdf := referenceecdffn(activediff)

	// 	Scale so max probability is in tails and prob at 0.5 is 0.
	return (2 * math.Abs(0.5-activeecdf)), nil
}

// Generates the percent difference between the means of the reference and active
// data. Returns a value scaled such that it lies between 0 and 1.
func Prob(vector govector.Vector) (float64, error) {
	totsize := vector.Len()
	reference := vector[0:(totsize - actsize)]
	active := vector[(totsize - actsize):totsize]

	activemean, _ := active.Mean()
	referencemean, _ := reference.Mean()

	if referencemean == 0 {
		return 0, fmt.Errorf("Percent diff formula will fail")
	}

	pdiff := (activemean - referencemean) / referencemean
	return 1 / (1 + math.Exp(-6*pdiff)), nil
}

// Implements the Kolmogorv-Smirnov test. The p-score returned is not consistent
// results obtained in R, but is consistent with results from the skyline package
// (https://github.com/etsy/skyline).
func KS(vector govector.Vector) (float64, error) {
	totsize := vector.Len()
	reference := vector[0:(totsize - actsize)]
	active := vector[(totsize - actsize):totsize]

	n1 := active.Len()
	n2 := reference.Len()
	n := 100

	// First sort the active data and generate a cummulative distribution function
	// using that data. Do the same for the reference data.
	sort.Sort(active)
	activeecdffn, _ := active.Ecdf()
	sort.Sort(reference)
	referenceecdffn, _ := reference.Ecdf()

	// We want the reference and active vectors to have the same length n, so we
	// consider the min and max for each and interpolated the points between.
	referenceinterp := interpolateCDF(reference, n)
	activeinterp := interpolateCDF(active, n)

	// Then we apply the distribution function over the interpolated data.
	activeecdf := activeinterp.Apply(activeecdffn)
	referenceecdf := referenceinterp.Apply(referenceecdffn)

	// Find the maximum displacement between both distributions. Use this value
	// to calculate the KS test score.
	d := 0.0
	for i := 0; i < n; i++ {
		d = math.Max(d, math.Abs(activeecdf[i]-referenceecdf[i]))
	}

	en := math.Sqrt(float64(n1*n2) / float64(n1+n2))
	prob := kolmogorov((en + 0.12 + 0.11/en) * d)
	return (1 - prob), nil
}

// A helper function for KS that rescales a vector to the desired length npoints.
func interpolateCDF(vector govector.Vector, npoints int) govector.Vector {
	interp := make(govector.Vector, npoints)
	max := vector.Max()
	min := vector.Min()
	step := (max - min) / (float64(npoints) - 1)
	interp[0] = min
	i := 1
	for i < npoints {
		interp[i] = interp[i-1] + step
		i++
	}
	return interp
}

// A helper function to calculate the KS test score. Grabbed from skyline.
func kolmogorov(y float64) float64 {
	if y < 1.1e-16 {
		return 1.0
	}
	x := -2.0 * y * y
	sign := 1.0
	p := 0.0
	r := 1.0
	var t float64
	for {
		t = math.Exp(x * r * r)
		p += sign * t
		if t == 0.0 {
			break
		}
		r += 1.0
		sign = -sign
		if (t / p) <= 1.1e-16 {
			break
		}
	}
	return (p + p)
}
