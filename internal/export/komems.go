package export

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/sysrqio/liegenpulse/internal/audit"
	"github.com/sysrqio/liegenpulse/internal/config"
)

// KomEMS CSV: simplified municipal EMS import layout
func KomEMS(root *config.Root, result *audit.Result) (string, error) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	w.Comma = ';'

	header := []string{
		"timestamp",
		"site",
		"building_id",
		"meter_id",
		"medium",
		"protocol",
		"endpoint",
		"reachable",
		"carrier",
		"readiness_score",
	}
	if err := w.Write(header); err != nil {
		return "", err
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	buildingScore := map[string]float64{}
	if result != nil {
		for _, bs := range result.Buildings {
			buildingScore[bs.BuildingID] = bs.Score
		}
	}

	meterReach := map[string]bool{}
	if result != nil {
		for _, m := range result.Meters {
			key := m.BuildingID + "/" + m.MeterID
			meterReach[key] = m.Reachable
		}
	}

	for _, b := range root.Buildings {
		bs := buildingScore[b.ID]
		for _, m := range b.Meters {
			endpoint := m.Address
			if endpoint == "" {
				endpoint = m.Topic
			}
			rel := "unknown"
			if result != nil && result.Simulated {
				if meterReach[b.ID+"/"+m.ID] {
					rel = "yes"
				} else {
					rel = "no"
				}
			}
			row := []string{
				ts,
				root.Site,
				b.ID,
				m.ID,
				string(m.Type),
				string(m.Protocol),
				endpoint,
				rel,
				m.Carrier,
				fmt.Sprintf("%.2f", bs),
			}
			if err := w.Write(row); err != nil {
				return "", err
			}
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}
