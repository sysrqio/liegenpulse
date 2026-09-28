package generate_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sysrqio/liegenpulse/internal/config"
	"github.com/sysrqio/liegenpulse/internal/generate"
)

func TestTelegrafContainsInputs(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "valid.yaml")
	root, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	out := generate.Telegraf(root)
	if !strings.Contains(out, "inputs.modbus") {
		t.Fatal("missing modbus section")
	}
}
