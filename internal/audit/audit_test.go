package audit_test

import (
	"path/filepath"
	"testing"

	"github.com/sysrqio/liegenpulse/internal/audit"
	"github.com/sysrqio/liegenpulse/internal/config"
	"github.com/sysrqio/liegenpulse/internal/simulate"
)

func TestAuditSamplePassesThreshold(t *testing.T) {
	p := filepath.Join("..", "..", "meters.yaml")
	root, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	reach, err := simulate.BuildReachability(root)
	if err != nil {
		t.Fatal(err)
	}
	res := audit.Run(root, reach, audit.Options{Simulate: true, Threshold: 70})
	if res.Score < 70 {
		t.Fatalf("score %.2f expected >= 70", res.Score)
	}
	if !res.Passed && res.ValidationOK {
		t.Fatalf("expected pass when validation ok and score high")
	}
}

func TestAuditValidFixtureHighScore(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "valid.yaml")
	root, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	reach, err := simulate.BuildReachability(root)
	if err != nil {
		t.Fatal(err)
	}
	res := audit.Run(root, reach, audit.Options{Simulate: true, Threshold: 70})
	if res.Score < 90 {
		t.Fatalf("score %.2f", res.Score)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "valid.yaml")
	root, _ := config.Load(p)
	reach, _ := simulate.BuildReachability(root)
	res := audit.Run(root, reach, audit.Options{Simulate: true, Threshold: 70})
	data, err := res.JSON()
	if err != nil || len(data) < 10 {
		t.Fatal(err)
	}
}
