package anomalize

import (
	"fmt"
	"github.com/drewlanenga/govector"
	"math"
	"math/rand"
	"sort"
	//	"code.google.com/p/gostat/stat"
)

// Generates a random walk given the number of steps desired, a starting point,
// and the desired standard deviation.
func RandomWalk(nsteps, start int, sd float64) (govector.Vector, error) {
	walk := make(govector.Vector, nsteps)

	walk[0] = float64(start)

	i := 1
	for i < nsteps {
		step := rand.NormFloat64() * sd
		walk[i] = walk[i-1] + step

		i++
	}
	return walk, nil
}

// This function should return a probability given z, but
// doesn't currently work due to the broken gostat pkg
//func ZtoP(z float64) float64 {
//	p := stat.Z_PDF()(z)
//	if (p > 0.5) {
//		p = 1 - p
//	}
//	return PtoProb(p/2)
//}

func PtoProb(p float64) (float64, error) {
	return (1 - p), nil
}

func Cap(x, xmin, xmax float64) (float64, error) {
	if xmin > xmax {
		return 0, fmt.Errorf("Xmin must be greater than or equal to Xmax")
	}
	return math.Max(math.Min(x, xmax), xmin), nil
}

// Generates permutations of reference and active window values to determine
// whether or not data is anomalous. Must be given reference and active data
// and the number of permutations desired.
func Rank(reference, active govector.Vector, nsteps int) (float64, error) {
	if nsteps == 0 {
		return 0, fmt.Errorf("Number of iterations must be greater than zero")
	}
	reflength := reference.Len()

	// Append the active data to the reference data, so that we have one bigger
	// vector containing both.
	vector, err := reference.Append(active)
	length := int(vector.Len())

	// Find the differences between neighboring elements and rank those differences.
	samp, _ = vector.Diff()
	samp, _ = samp.Rank()

	// Consider the sum of the ranks across the active data. This is the sum that
	// we will compare our permutations to. (The indexing runs to length-1 because
	// after applying .Diff(), we have decreased the length of out vector by 1.)
	summ := samp[reflength : length-1].Sum()
	i := 0
	count := 0
	tempsumm := 0.0

	// Permute the active and reference data and compute the sums across the tail
	// (from the length of the reference data to the full length).
	for i < nsteps {
		temp, _ := vector.Sample(length)
		temp, _ = temp.Diff()
		temp, _ = temp.Rank()

		tempsumm = temp[reflength : length-1].Sum()

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
	return float64(count) / float64(nsteps), err
}

// Generates the cumulative distribution function using the difference in the means
// for the active and reference data.
func DiffCDF(reference, active govector.Vector) (float64, error) {
	n := active.Len()
	m := reference.Len()

	if m < 2 || n < 2 {
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
	return 2 * math.Abs(0.5-activeecdf), nil
}

// Generates the percent difference between the means of the reference and active
// data. Returns a value scaled such that it lies between 0 and 1.
func Prob(reference, active govector.Vector) (float64, error) {
	activemean, _ := active.Mean()
	referencemean, _ := reference.Mean()

	if referencemean == 0 {
		return 0, fmt.Errorf("Percent diff formula will fail")
	}

	pdiff := (activemean - referencemean) / referencemean
	return 1 / (1 + math.Exp(-6*pdiff)), nil
}

// This function does not work because the gostat pkg is broken.
//func ThreeSigma(reference, active govector.Vector) (float64, error) {
//	activemean, _ := active.Mean()
//	referencemean, _ := reference.Mean()
//	referencesd, _ := reference.Sd()
//
//	if referencesd == 0 {
//		return 0, fmt.Errorf("Calculating z will fail")
//	}
//
//	z := (activemean - referencemean) / referencesd
//	return ZtoP(z), nil
//}

// Returns the Kolmogorov-Smirnov test given the active and reference data.
func KS(reference, active govector.Vector) (float64, error) {
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
