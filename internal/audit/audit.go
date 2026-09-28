package audit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sysrqio/liegenpulse/internal/config"
	"github.com/sysrqio/liegenpulse/internal/simulate"
)

type MeterStatus struct {
	BuildingID string `json:"building_id"`
	MeterID    string `json:"meter_id"`
	Type       string `json:"type"`
	Protocol   string `json:"protocol"`
	Reachable  bool   `json:"reachable"`
	CarrierOK  bool   `json:"carrier_ok"`
}

type BuildingScore struct {
	BuildingID   string  `json:"building_id"`
	Score        float64 `json:"score"`
	Coverage     float64 `json:"coverage_pct"`
	Reachability float64 `json:"reachability_pct"`
	Carrier      float64 `json:"carrier_pct"`
}

type Result struct {
	Site           string          `json:"site"`
	Score          float64         `json:"score"`
	Threshold      int             `json:"threshold"`
	Passed         bool            `json:"passed"`
	Simulated      bool            `json:"simulated"`
	Buildings      []BuildingScore `json:"buildings"`
	Meters         []MeterStatus   `json:"meters"`
	ValidationOK   bool            `json:"validation_ok"`
	IssueCount     int             `json:"issue_count"`
}

type Options struct {
	Simulate  bool
	Threshold int
}

func Run(root *config.Root, reach simulate.Reachability, opts Options) Result {
	issues := root.Validate()
	res := Result{
		Site:         root.Site,
		Threshold:    opts.Threshold,
		Simulated:    opts.Simulate,
		ValidationOK: len(issues) == 0,
		IssueCount:   len(issues),
	}

	var buildingScores []BuildingScore
	var meterStatuses []MeterStatus
	var totalWeight float64
	var weightedSum float64

	for _, b := range root.Buildings {
		bs, meters := scoreBuilding(b, reach, opts.Simulate)
		buildingScores = append(buildingScores, bs)
		meterStatuses = append(meterStatuses, meters...)
		w := buildingWeight(b)
		totalWeight += w
		weightedSum += bs.Score * w
	}

	if totalWeight == 0 {
		res.Score = 0
	} else {
		res.Score = round2(weightedSum / totalWeight)
	}

	// Penalize validation issues (schema / EnEfG completeness)
	if len(issues) > 0 {
		penalty := float64(len(issues)) * 5
		if penalty > 40 {
			penalty = 40
		}
		res.Score = round2(max(0, res.Score-penalty))
	}

	res.Passed = res.ValidationOK && res.Score >= float64(opts.Threshold)
	res.Buildings = buildingScores
	res.Meters = meterStatuses
	return res
}

func buildingWeight(b config.Building) float64 {
	n := len(b.Meters)
	if n == 0 {
		return 1
	}
	return float64(n)
}

func scoreBuilding(b config.Building, reach simulate.Reachability, simulated bool) (BuildingScore, []MeterStatus) {
	typesNeeded := []config.MeterType{config.MeterElectricity}
	if b.HasHeat || hasType(b, config.MeterHeat) {
		typesNeeded = append(typesNeeded, config.MeterHeat)
	}

	present := map[config.MeterType]bool{}
	for _, m := range b.Meters {
		present[m.Type] = true
	}
	coverageHits := 0
	for _, t := range typesNeeded {
		if present[t] {
			coverageHits++
		}
	}
	coveragePct := pct(coverageHits, len(typesNeeded))

	var reachable, carrierOK, total int
	var statuses []MeterStatus
	for _, m := range b.Meters {
		total++
		rel := false
		if simulated {
			rel = simulate.IsReachable(reach, b, m)
		}
		cOK := strings.TrimSpace(m.Carrier) != ""
		if rel {
			reachable++
		}
		if cOK {
			carrierOK++
		}
		statuses = append(statuses, MeterStatus{
			BuildingID: b.ID,
			MeterID:    m.ID,
			Type:       string(m.Type),
			Protocol:   string(m.Protocol),
			Reachable:  rel,
			CarrierOK:  cOK,
		})
	}

	reachPct := 100.0
	if simulated && total > 0 {
		reachPct = pct(reachable, total)
	}
	carrierPct := 100.0
	if total > 0 {
		carrierPct = pct(carrierOK, total)
	}

	// Weights: coverage 40%, reachability 40%, carrier 20%
	score := coveragePct*0.4 + reachPct*0.4 + carrierPct*0.2
	return BuildingScore{
		BuildingID:   b.ID,
		Score:        round2(score),
		Coverage:     round2(coveragePct),
		Reachability: round2(reachPct),
		Carrier:      round2(carrierPct),
	}, statuses
}

func hasType(b config.Building, t config.MeterType) bool {
	for _, m := range b.Meters {
		if m.Type == t {
			return true
		}
	}
	return false
}

func pct(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (r Result) FormatHuman(verbose bool) string {
	var sb strings.Builder
	status := "FAIL"
	if r.Passed {
		status = "PASS"
	}
	fmt.Fprintf(&sb, "EnEfG readiness audit — %s\n", r.Site)
	fmt.Fprintf(&sb, "Score: %.2f / 100 (threshold %d) [%s]\n", r.Score, r.Threshold, status)
	if r.Simulated {
		sb.WriteString("Mode: simulate (embedded fixture, no network)\n")
	}
	if r.IssueCount > 0 {
		fmt.Fprintf(&sb, "Validation issues: %d (score penalized)\n", r.IssueCount)
	}
	for _, bs := range r.Buildings {
		fmt.Fprintf(&sb, "  building %s: score %.2f (coverage %.0f%%, reach %.0f%%, carrier %.0f%%)\n",
			bs.BuildingID, bs.Score, bs.Coverage, bs.Reachability, bs.Carrier)
	}
	if verbose {
		sb.WriteString("\nMeters:\n")
		for _, m := range r.Meters {
			fmt.Fprintf(&sb, "  %s/%s %s/%s reachable=%v carrier_ok=%v\n",
				m.BuildingID, m.MeterID, m.Type, m.Protocol, m.Reachable, m.CarrierOK)
		}
	}
	return sb.String()
}

func (r Result) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
