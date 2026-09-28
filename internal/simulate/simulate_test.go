package simulate_test

import (
	"path/filepath"
	"testing"

	"github.com/sysrqio/liegenpulse/internal/config"
	"github.com/sysrqio/liegenpulse/internal/simulate"
)

func TestFixtureOfflineHeat(t *testing.T) {
	p := filepath.Join("..", "..", "meters.yaml")
	root, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	reach, err := simulate.BuildReachability(root)
	if err != nil {
		t.Fatal(err)
	}
	key := simulate.MeterKey("office-a", "heat-a")
	if reach[key] {
		t.Fatal("office-a heat-a should be unreachable in fixture")
	}
}

func TestLoadFixtureNoNetwork(t *testing.T) {
	r, err := simulate.LoadFixture()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) == 0 {
		t.Fatal("empty fixture")
	}
}
