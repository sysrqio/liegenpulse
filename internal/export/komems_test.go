package export_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sysrqio/liegenpulse/internal/audit"
	"github.com/sysrqio/liegenpulse/internal/config"
	"github.com/sysrqio/liegenpulse/internal/export"
	"github.com/sysrqio/liegenpulse/internal/simulate"
)

func TestKomEMSCSVHeader(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "valid.yaml")
	root, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	reach, _ := simulate.BuildReachability(root)
	res := audit.Run(root, reach, audit.Options{Simulate: true, Threshold: 70})
	csv, err := export.KomEMS(root, &res)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(csv, "timestamp;") {
		t.Fatalf("header: %q", csv[:30])
	}
	if !strings.Contains(csv, "electricity") {
		t.Fatal("missing meter row")
	}
}
