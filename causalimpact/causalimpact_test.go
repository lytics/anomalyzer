package causalimpact

import (
	"github.com/bmizerany/assert"
	"github.com/drewlanenga/govector"
	"testing"
)

func TestCausalImpact(t *testing.T) {
	conf := &CausalImpactConf{
		ActiveSize: 1,
	}

	data := []float64{0.1, 2.05, 1.5, 2.5, 2.6, 2.55, 2.15}
	vector, _ := govector.AsVector(data)
	x := CausalImpactStruct{conf, vector}

	_, boo, err := x.CausalImpact()
	assert.Equal(t, nil, err, "Error generating causal impact score")
	assert.Equal(t, boo, false, "False positive")

	// add an anomalous point
	vector[len(data)-1] = 22
	x = CausalImpactStruct{conf, vector}
	_, boo, err = x.CausalImpact()
	assert.Equal(t, nil, err, "Error generating causal impact score")
	assert.Equal(t, boo, true, "False negative")
}
