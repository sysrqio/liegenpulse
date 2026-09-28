package simulate

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sysrqio/liegenpulse/internal/config"
)

//go:embed fixtures/reachability.json
var fixtureData embed.FS

type Reachability map[string]bool

func LoadFixture() (Reachability, error) {
	raw, err := fixtureData.ReadFile("fixtures/reachability.json")
	if err != nil {
		return nil, err
	}
	var r Reachability
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func MeterKey(buildingID, meterID string) string {
	return buildingID + "/" + meterID
}

func IsReachable(r Reachability, b config.Building, m config.Meter) bool {
	if r == nil {
		return false
	}
	keys := []string{
		MeterKey(b.ID, m.ID),
		m.ID,
	}
	if m.Protocol == config.ProtocolModbus && m.Address != "" {
		keys = append(keys, m.Address, b.ID+"/"+m.Address)
	}
	if m.Protocol == config.ProtocolMQTT && m.Topic != "" {
		keys = append(keys, m.Topic, b.ID+"/"+m.Topic)
	}
	for _, k := range keys {
		if v, ok := r[k]; ok {
			return v
		}
	}
	// Default: reachable if address/topic looks like fixture pattern
	ref := m.Address
	if ref == "" {
		ref = m.Topic
	}
	ref = strings.ToLower(ref)
	if strings.Contains(ref, "unreachable") || strings.HasSuffix(ref, "/offline") {
		return false
	}
	return true
}

func BuildReachability(root *config.Root) (Reachability, error) {
	fix, err := LoadFixture()
	if err != nil {
		return nil, fmt.Errorf("load simulate fixture: %w", err)
	}
	out := make(Reachability)
	for _, b := range root.Buildings {
		for _, m := range b.Meters {
			key := MeterKey(b.ID, m.ID)
			out[key] = IsReachable(fix, b, m)
		}
	}
	return out, nil
}
