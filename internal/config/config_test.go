package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sysrqio/liegenpulse/internal/config"
)

func TestLoadValid(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "valid.yaml")
	root, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if root.Site != "test-site" {
		t.Fatalf("site=%q", root.Site)
	}
	issues := root.Validate()
	for _, iss := range issues {
		if strings.Contains(iss.Message, "EnEfG") {
			t.Fatalf("unexpected issue: %+v", iss)
		}
	}
}

func TestLoadInvalidAddress(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "invalid.yaml")
	root, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	issues := root.Validate()
	if len(issues) == 0 {
		t.Fatal("expected validation issues")
	}
}

func TestEnEfGMissingElectricity(t *testing.T) {
	root := &config.Root{
		Site: "s",
		Buildings: []config.Building{{
			ID: "b1",
			Meters: []config.Meter{{
				ID: "w1", Type: config.MeterWater, Protocol: config.ProtocolMQTT, Topic: "t/w",
			}},
		}},
	}
	issues := root.Validate()
	found := false
	for _, iss := range issues {
		if strings.Contains(iss.Message, "electricity") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected electricity requirement issue")
	}
}
