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

func TestTelegrafMQTTPath(t *testing.T) {
	p := filepath.Join("..", "..", "meters.yaml")
	root, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	out := generate.Telegraf(root)
	if !strings.Contains(out, "[[inputs.mqtt_consumer]]") {
		t.Fatal("missing mqtt_consumer block")
	}
	if !strings.Contains(out, "topic_parsing") {
		t.Fatal("missing topic_parsing for mqtt meters")
	}
	if !strings.Contains(out, "municipality/meters/main-hall/water") {
		t.Fatal("expected water mqtt topic in telegraf output")
	}
}
