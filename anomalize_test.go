package anomalize

import (
	"fmt"
	"github.com/bmizerany/assert"
	"math"
	"testing"
)

func TestRandomWalk(t *testing.T) {
	walk, err := RandomWalk(10, 0.5, 0.05)
	fmt.Println(walk)

	assert.Equal(t, nil, err, "Error generating random walk")
}

func TestPtoProb(t *testing.T) {
	prob, err := PtoProb(0.25)
	fmt.Println(prob)

	assert.Equal(t, nil, err, "Error in calculating prob given p")
}

func TestCap(t *testing.T) {
	capp, err := Cap(200.0, 0.0, 100.0)

	assert.Equal(t, nil, err, "Error in calculating cap")
	assert.Equal(t, 100.0, capp, "Error in calculating cap")
}

func TestRank(t *testing.T) {
	vector, err := RandomWalk(100, 0.5, 0.0)
	score, err := Rank(vector)

	// For no standard deviation, the random walk should stand still and
	// a rank test will return 0. For any other standard deviations, the
	// result of the rank test is somewhat random.
	assert.Equal(t, nil, err, "Error in calculating rank")
	assert.Equal(t, score, 0.0, "Error in calculating rank")
}

func TestDiffCDF(t *testing.T) {
	vector, err := RandomWalk(20, 0.2, 0.05)
	score, err := DiffCDF(vector)
	fmt.Println(score)

	assert.Equal(t, nil, err, "Error in calculating diffCDF")
}

func TestProb(t *testing.T) {
	vector, err := RandomWalk(20, 0.5, 0.9)
	score, err := Prob(vector)
	fmt.Println(score)

	assert.Equal(t, nil, err, "Error in calculating percent diff")
}

func TestKS(t *testing.T) {
	// A random walk with no standard deviation will return a KS score
	// of zero.
	vector, err := RandomWalk(100, 10000, 0.0)
	score, err := KS(vector)
	score = math.Floor(score)

	assert.Equal(t, 0.0, score, "Error in calculating KS score")
	assert.Equal(t, nil, err, "Error in calculating KS score")
}

func TestBounds(t *testing.T) {
	// A random walk around 0.5 should return a relatively low bounds
	// score, since the larger the deviation from the mean of the upper
	// and lower bounds (0.5 in this case), the larger the result of
	// bounds is.
	vector, err := RandomWalk(20, 0.5, 0.005)
	score, err := Bounds(vector)
	fmt.Println(score)

	assert.Equal(t, nil, err, "Error in calculating bounds score")

	// Here's a random walk that will return a higher bounds score since
	// it hovers around the lower bound.
	vector, err = RandomWalk(20, 0.05, 0.005)
	score, err = Bounds(vector)
	fmt.Println(score)

	assert.Equal(t, nil, err, "Error in calculating bounds score")
}
