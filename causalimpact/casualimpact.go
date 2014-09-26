package causalimpact

import (
	"encoding/json"
	"github.com/drewlanenga/govector"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

type CausalImpactConf struct {
	ActiveSize int
}

type CausalImpactStruct struct {
	Conf *CausalImpactConf
	Data govector.Vector
}

// Implementation of github.com/google/CausalImpact code
func (a CausalImpactStruct) CausalImpact() (float64, bool, error) {
	// convert the data and window size to strings for the
	// command line
	datastr := make([]string, len(a.Data))
	datastr[0] = strconv.FormatFloat(a.Data[0], 'f', 4, 64)
	i := 1
	for i < len(a.Data) {
		datastr[i] = strconv.FormatFloat(a.Data[i], 'f', 3, 64)
		i++
	}
	// join each string with a comma between them
	datastring := strings.Join(datastr, ",")
	timestring := strconv.Itoa(a.Conf.ActiveSize)

	// execute the R script which runs "Causal Impact"
	out, err := exec.Command("./impact.r", datastring, timestring).Output()
	if err != nil {
		return math.NaN(), false, err
	}

	// define a struct which will include the JSON outputs
	// of that R script
	type Routput struct {
		Lower float64 `json:"lower"`
		Upper float64 `json:"upper"`
		P     float64 `json:"p"`
	}

	// unmarshal the JSON outputs
	var routputs Routput
	err = json.Unmarshal(out, &routputs)
	if err != nil {
		return math.NaN(), false, err
	}

	// return the "posterior probability of causal effect", and
	// a boolean corresponding to whether or not the range of
	// the lower and upper averages do not cross zero.
	if routputs.Upper > 0 {
		if routputs.Lower < 0 {
			return (1 - routputs.P), false, nil
		} else {
			return (1 - routputs.P), true, nil
		}
	}
	// if the upper bound is below zero, then the lower bound is
	// below it and they do not cross zero
	if routputs.Upper < 0 {
		return (1 - routputs.P), true, nil
	}

	return (1 - routputs.P), false, nil
}
