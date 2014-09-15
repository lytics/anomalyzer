package anomalize

import (
	"github.com/drewlanenga/govector"
	"math"
	"sort"
)

type Algorithm func(govector.Vector, AnomalizerConf) float64

var (
	Algorithms = map[string]Algorithm{
		"magnitude": MagnitudeTest,
		"rank":      RankTest,
		"diff":      DiffCDFTest,
		"fence":     FenceTest,
		"ks":        KSTest,
	}
)

// Identity function
func identity(anything interface{}) interface{} {
	return anything
}

// Returns a value within a given window (xmin and xmax).
func cap(x, min, max float64) float64 {
	return math.Max(math.Min(x, max), min)
}

// Returns a contant
func constant(x float64) float64 {
	return 0.2
}

// Return integer math comparisons
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func min(x, y int) int {
	if y < x {
		return y
	}
	return x
}

// Return a vector slice for the active window
func extractWindows(vector govector.Vector, refSize, activeSize int) (govector.Vector, govector.Vector) {
	n := len(vector)
	activeSize = min(activeSize, n)
	refSize = min(refSize, n-activeSize)

	// return reference and active windows
	return vector[n-activeSize-refSize : n-activeSize], vector[n-activeSize:]
}

// This function can be used to test whether or not data is getting close to a
// specified upper or lower bound.
func FenceTest(vector govector.Vector, conf AnomalizerConf) float64 {
	// we don't really care about a reference window for this one
	_, active := extractWindows(vector, conf.ReferenceSize, conf.ActiveSize)

	x := active.Mean()

	distance := 0.0
	if conf.LowerBound == NA {
		// we only care about distance from the upper bound
		distance = x / conf.UpperBound
	} else {
		// we care about both bounds, so measure distance
		// from midpoint

		bound := (conf.UpperBound - conf.LowerBound) / 2
		mid := conf.LowerBound + bound

		distance = (math.Abs(x - mid)) / bound
	}
	return weightExp(cap(distance, 0, 1), 10)
}

// This is a function will sharply scale values between 0 and 1 such that
// smaller values are weighted more towards 0. A larger base value means a
// more horshoe type function.
func weightExp(x, base float64) float64 {
	return (math.Pow(base, x) - 1) / (math.Pow(base, 1) - 1)
}

// Generates permutations of reference and active window values to determine
// whether or not data is anomalous. The number of permutations desired has
// been set to 500 but can be increased for more precision.
func RankTest(vector govector.Vector, conf AnomalizerConf) float64 {
	// Find the differences between neighboring elements and rank those differences.
	ranks := vector.Diff().Apply(math.Abs).Rank()

	// The indexing runs to length-1 because after applying .Diff(), We have
	// decreased the length of out vector by 1.
	_, active := extractWindows(ranks, conf.ReferenceSize-1, conf.ActiveSize)

	// Consider the sum of the ranks across the active data. This is the sum that
	// we will compare our permutations to.
	activeSum := active.Sum()

	i := 0
	significant := 0

	// Permute the active and reference data and compute the sums across the tail
	// (from the length of the reference data to the full length).
	for i < conf.PermCount {
		permRanks := vector.Shuffle().Diff().Apply(math.Abs).Rank()
		_, permActive := extractWindows(permRanks, conf.ReferenceSize-1, conf.ActiveSize)

		// If we find a sum that is less than the initial sum across the active data,
		// this implies our initial sum might be uncharacteristically high. We increment
		// our count.
		if permActive.Sum() < activeSum {
			significant++
		}
		i++
	}
	// We return the percentage of the number of iterations where we found our initial
	// sum to be high.
	return float64(significant) / float64(conf.PermCount)
}

// Generates the cumulative distribution function using the difference in the means
// for the data.
func DiffCDFTest(vector govector.Vector, conf AnomalizerConf) float64 {
	diffs := vector.Diff().Apply(math.Abs)
	reference, active := extractWindows(diffs, conf.ReferenceSize-1, conf.ActiveSize)

	// Find the empircal distribution function using the reference window.
	refEcdf := reference.Ecdf()

	// Difference between the active and reference means.
	activeDiff := active.Mean() - reference.Mean()

	// Apply the empirical distribution function to that difference.
	percentile := refEcdf(activeDiff)

	// Scale so max probability is in tails and prob at 0.5 is 0.
	return (2 * math.Abs(0.5-percentile))
}

// Generates the percent difference between the means of the reference and active
// data. Returns a value scaled such that it lies between 0 and 1.
func MagnitudeTest(vector govector.Vector, conf AnomalizerConf) float64 {
	reference, active := extractWindows(vector, conf.ReferenceSize, conf.ActiveSize)

	activeMean := active.Mean()
	refMean := reference.Mean()

	// If the baseline is 0, then the magnitude should be Inf, but we'll
	// round to 1.
	if refMean == 0 {
		return 1
	}

	pdiff := (activeMean - refMean) / refMean
	return weightExp(pdiff, 10)
}

// Implements the Kolmogorv-Smirnov test. The p-score returned is not consistent
// results obtained in R, but is consistent with results from the skyline package
// (https://github.com/etsy/skyline).
func KSTest(vector govector.Vector, conf AnomalizerConf) float64 {
	reference, active := extractWindows(vector, conf.ReferenceSize, conf.ActiveSize)

	n1 := len(reference)
	n2 := len(active)

	// First sort the active data and generate a cummulative distribution function
	// using that data. Do the same for the reference data.
	sort.Sort(active)
	activeEcdf := active.Ecdf()
	sort.Sort(reference)
	refEcdf := reference.Ecdf()

	// We want the reference and active vectors to have the same length n, so we
	// consider the min and max for each and interpolated the points between.
	refInterp := interpolate(reference, n1+n2)
	activeInterp := interpolate(active, n1+n2)

	// Then we apply the distribution function over the interpolated data.
	activeDist := activeInterp.Apply(activeEcdf)
	refDist := refInterp.Apply(refEcdf)

	// Find the maximum displacement between both distributions. Use this value
	// to calculate the KS test score.
	d := 0.0
	for i := 0; i < n1+n2; i++ {
		d = math.Max(d, math.Abs(activeDist[i]-refDist[i]))
	}

	en := math.Sqrt(float64(n1*n2) / float64(n1+n2))
	prob := kolmogorov((en + 0.12 + 0.11/en) * d)
	return (1 - prob)
}

// A helper function for KS that rescales a vector to the desired length npoints.
func interpolate(vector govector.Vector, npoints int) govector.Vector {
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

// A helper function to calculate the KS test statistic.
// Reference: scipy/special/cephes/kolmogorov.c
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
