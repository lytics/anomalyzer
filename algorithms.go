package anomalyzer

import (
	"fmt"
	"github.com/drewlanenga/govector"
	"math"
)

type Algorithm func(govector.Vector, AnomalyzerConf) float64

var (
	Algorithms = map[string]Algorithm{
		"magnitude": MagnitudeTest,
		"diff":      DiffTest,
		"highrank":  RankTest,
		"lowrank":   ReverseRankTest,
		"cdf":       CDFTest,
		"fence":     FenceTest,
		"ks":        BootstrapKsTest,
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

// Return a vector slice for the active window and reference window.
// Some tests require different minimum thresholds for sizes of reference windows.
// This can be specified in the minRefSize parameter. If size isn't important, use -1
func extractWindows(vector govector.Vector, refSize, activeSize, minRefSize int) (govector.Vector, govector.Vector, error) {
	n := len(vector)
	activeSize = min(activeSize, n)
	refSize = min(refSize, n-activeSize)

	// make sure the reference size is at least as big as the active size
	// note that this penalty might be overly severe for some tests
	if refSize < minRefSize {
		return nil, nil, fmt.Errorf("Reference size must be at least as big as active size")
	}

	// return reference and active windows
	return vector[n-activeSize-refSize : n-activeSize], vector[n-activeSize:], nil
}

// This function can be used to test whether or not data is getting close to a
// specified upper or lower bound.
func FenceTest(vector govector.Vector, conf AnomalyzerConf) float64 {
	// we don't really care about a reference window for this one
	_, active, _ := extractWindows(vector, conf.referenceSize, conf.ActiveSize, -1)

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
func DiffTest(vector govector.Vector, conf AnomalyzerConf) float64 {
	// Find the differences between neighboring elements and rank those differences.
	ranks := vector.RelDiff().Apply(math.Abs).Rank()

	// The indexing runs to length-1 because after applying .Diff(), We have
	// decreased the length of out vector by 1.
	_, active, err := extractWindows(ranks, conf.referenceSize-1, conf.ActiveSize, conf.ActiveSize)
	if err != nil {
		return NA
	}

	// Consider the sum of the ranks across the active data. This is the sum that
	// we will compare our permutations to.
	activeSum := active.Sum()

	i := 0
	significant := 0

	// Permute the active and reference data and compute the sums across the tail
	// (from the length of the reference data to the full length).
	for i < conf.PermCount {
		permRanks := vector.Shuffle().RelDiff().Apply(math.Abs).Rank()
		_, permActive, _ := extractWindows(permRanks, conf.referenceSize-1, conf.ActiveSize, conf.ActiveSize)

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

func RankTest(vector govector.Vector, conf AnomalyzerConf) float64 {
	return rankTest(vector, conf, lessThan)
}

func ReverseRankTest(vector govector.Vector, conf AnomalyzerConf) float64 {
	return rankTest(vector, conf, greaterThan)
}

type compare func(x, y float64) bool

func greaterThan(x, y float64) bool {
	if x > y {
		return true
	}
	return false
}

func lessThan(x, y float64) bool {
	if x < y {
		return true
	}
	return false
}

// Generates permutations of reference and active window values to determine
// whether or not data is anomalous. The number of permutations desired defaults
// to 500 but can be increased for more precision. A comparison function above
// can be specified to create Rank and ReverseRank tests.
func rankTest(vector govector.Vector, conf AnomalyzerConf, comparison compare) float64 {
	// Rank the elements of a vector
	ranks := vector.Rank()

	_, active, err := extractWindows(ranks, conf.referenceSize, conf.ActiveSize, conf.ActiveSize)
	if err != nil {
		return NA
	}

	// Consider the sum of the ranks across the active data. This is the sum that
	// we will compare our permutations to.
	activeSum := active.Sum()

	i := 0
	significant := 0

	// Permute the active and reference data and compute the sums across the tail
	// (from the length of the reference data to the full length).
	for i < conf.PermCount {
		permRanks := vector.Shuffle().Rank()
		_, permActive, _ := extractWindows(permRanks, conf.referenceSize, conf.ActiveSize, conf.ActiveSize)

		// If we find a sum that is less than the initial sum across the active data,
		// this implies our initial sum might be uncharacteristically high. We increment
		// our count.

		permSum := permActive.Sum()
		if comparison(permSum, activeSum) {
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
func CDFTest(vector govector.Vector, conf AnomalyzerConf) float64 {
	diffs := vector.Diff().Apply(math.Abs)
	reference, active, err := extractWindows(diffs, conf.referenceSize-1, conf.ActiveSize, conf.ActiveSize)
	if err != nil {
		return NA
	}

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
func MagnitudeTest(vector govector.Vector, conf AnomalyzerConf) float64 {
	reference, active, err := extractWindows(vector, conf.referenceSize, conf.ActiveSize, 1)
	if err != nil {
		return NA
	}

	activeMean := active.Mean()
	refMean := reference.Mean()

	// If the baseline is 0, then the magnitude should be Inf, but we'll
	// round to 1.
	if refMean == 0 {
		if activeMean == 0 {
			return 0
		} else {
			return 1
		}
	}
	pdiff := math.Abs(activeMean-refMean) / refMean
	//return weightExp(pdiff, 10)
	return pdiff
}

// Calculate a Kolmogorov-Smirnov test statistic.
func KsStat(vector govector.Vector, conf AnomalyzerConf) float64 {
	reference, active, err := extractWindows(vector, conf.referenceSize, conf.ActiveSize, conf.ActiveSize)
	if err != nil {
		return NA
	}
	n1 := len(reference)
	n2 := len(active)
	if n1%n2 != 0 {
		return NA
	}

	// First sort the active data and generate a cummulative distribution function
	// using that data. Do the same for the reference data.
	activeEcdf := active.Ecdf()
	refEcdf := reference.Ecdf()

	// We want the reference and active vectors to have the same length n, so we
	// consider the min and max for each and interpolated the points between.
	min := math.Min(reference.Min(), active.Min())
	max := math.Max(reference.Max(), active.Max())

	interpolated := interpolate(min, max, n1+n2)

	// Then we apply the distribution function over the interpolated data.
	activeDist := interpolated.Apply(activeEcdf)
	refDist := interpolated.Apply(refEcdf)

	// Find the maximum displacement between both distributions.
	d := 0.0
	for i := 0; i < n1+n2; i++ {
		d = math.Max(d, math.Abs(activeDist[i]-refDist[i]))
	}
	return d
}

func BootstrapKsTest(vector govector.Vector, conf AnomalyzerConf) float64 {
	dist := KsStat(vector, conf)
	if dist == NA {
		return NA
	}

	i := 0
	significant := 0

	for i < conf.PermCount {
		permVector := vector.Shuffle()
		permDist := KsStat(permVector, conf)

		if permDist < dist {
			significant++
		}
		i++
	}
	return float64(significant) / float64(conf.PermCount)
}

// A helper function for KS that rescales a vector to the desired length npoints.
func interpolate(min, max float64, npoints int) govector.Vector {
	interp := make(govector.Vector, npoints)

	step := (max - min) / (float64(npoints) - 1)
	interp[0] = min
	i := 1
	for i < npoints {
		interp[i] = interp[i-1] + step
		i++
	}
	return interp
}
