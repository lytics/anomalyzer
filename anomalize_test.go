package anomalize

import (
	"fmt"
	"github.com/bmizerany/assert"
	"testing"
)

func TestRandomWalk(t *testing.T) {

	walk, err := RandomWalk(10, 1000, 50.0)
	fmt.Println(walk)

	assert.Equal(t, nil, err, "Error generating random walk")
}

func TestPtoProb(t *testing.T) {
	prob, err := PtoProb(0.5)
	fmt.Println(prob)

	assert.Equal(t, nil, err, "Error in calculating prob given p")
}

func TestCap(t *testing.T) {
	capp, err := Cap(10, 0, 100)
	fmt.Println(capp)

	assert.Equal(t, nil, err, "Error in calculating cap")
}

func TestRank(t *testing.T) {
	active, err := RandomWalk(5, 100, 50.0)
	reference, err := RandomWalk(5, 100, 50.0)

	rank, err := Rank(reference, active, 10)

	assert.Equal(t, nil, err, "Error in calculating rank")
}

func TestDiffCDF(t *testing.T) {
	active, err := RandomWalk(100, 1, 50.0)
	reference, err := RandomWalk(100, 1000, 50.0)
	val, err := DiffCDF(reference, active)
	fmt.Println(val)

	assert.Equal(t, nil, err, "Error in calculating diffCDF")
}

func TestProb(t *testing.T) {
	active, err := RandomWalk(100, 1, 10.0)
	reference, err := RandomWalk(100, 1, 10.0)
	prob, err := Prob(active, reference)
	fmt.Println(prob)

	assert.Equal(t, nil, err, "Error in calculating percent diff")
}

//func TestThreeSigma(t *testing.T) {
//	active, err := RandomWalk(100, 1000, 10.0)
//	reference, err := RandomWalk(100, 1, 10.0)
//	z, err := ThreeSigma(active, reference)
//	fmt.Println(z)
//
//	assert.Equal(t, nil, err, "Error in calculating three sigma")
//}

func TestKS(t *testing.T) {
	active, err := RandomWalk(100, 10000, 0)
	reference, err := RandomWalk(100, 10000, 0)
	score, err := KS(active, reference)
	fmt.Println(score)

	assert.Equal(t, nil, err, "Error in calculating KS score")
}
