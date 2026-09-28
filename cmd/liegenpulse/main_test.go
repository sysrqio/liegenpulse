package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sysrqio/liegenpulse/internal/audit"
)

var cliBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "liegenpulse-cli-*")
	if err != nil {
		os.Stderr.WriteString("mkdir temp: " + err.Error() + "\n")
		os.Exit(1)
	}
	cliBin = filepath.Join(dir, "liegenpulse")
	build := exec.Command("go", "build", "-o", cliBin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		os.Stderr.WriteString(string(out))
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func runCLI(t *testing.T, root string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(cliBin, args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	combined := string(out)
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %v: %v\n%s", args, err, combined)
		}
		exitCode = ee.ExitCode()
	}
	return combined, "", exitCode
}

func TestAuditExitCodeSuccess(t *testing.T) {
	root := repoRoot(t)
	cfg := filepath.Join(root, "testdata", "valid.yaml")
	_, _, code := runCLI(t, root, "audit", "--config", cfg, "--simulate", "--threshold", "70")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestAuditExitCodeInvalidConfig(t *testing.T) {
	root := repoRoot(t)
	cfg := filepath.Join(root, "testdata", "invalid.yaml")
	_, _, code := runCLI(t, root, "audit", "--config", cfg, "--simulate")
	if code != 1 {
		t.Fatalf("expected exit 1 for invalid config, got %d", code)
	}
}

func TestAuditExitCodeBelowThreshold(t *testing.T) {
	root := repoRoot(t)
	cfg := filepath.Join(root, "meters.yaml")
	_, _, code := runCLI(t, root, "audit", "--config", cfg, "--simulate", "--threshold", "95")
	if code != 2 {
		t.Fatalf("expected exit 2 when score < threshold, got %d", code)
	}
}

func TestAuditJSONOutputFields(t *testing.T) {
	root := repoRoot(t)
	cfg := filepath.Join(root, "testdata", "valid.yaml")
	out, _, code := runCLI(t, root, "audit", "--config", cfg, "--simulate", "--output", "json", "--threshold", "70")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var res audit.Result
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if res.Site != "test-site" {
		t.Fatalf("site=%q", res.Site)
	}
	if res.Threshold != 70 {
		t.Fatalf("threshold=%d", res.Threshold)
	}
	if !res.Simulated {
		t.Fatal("expected simulated=true")
	}
	if len(res.Buildings) == 0 || len(res.Meters) == 0 {
		t.Fatalf("buildings/meters empty: %+v", res)
	}
	if !res.ValidationOK {
		t.Fatal("expected validation_ok")
	}
}

func TestVersionSmoke(t *testing.T) {
	root := repoRoot(t)
	out, _, code := runCLI(t, root, "version")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "liegenpulse") || !strings.Contains(out, "0.1.0") {
		t.Fatalf("version output: %q", out)
	}
}

func TestGenerateIncludesMQTTTelegraf(t *testing.T) {
	root := repoRoot(t)
	cfg := filepath.Join(root, "meters.yaml")
	out, _, code := runCLI(t, root, "generate", "--config", cfg)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "inputs.modbus") {
		t.Fatal("missing modbus telegraf section")
	}
	if !strings.Contains(out, "inputs.mqtt_consumer") {
		t.Fatal("missing mqtt_consumer telegraf section")
	}
	if !strings.Contains(out, "municipality/meters/office-a/heat") {
		t.Fatal("missing mqtt topic from config")
	}
}

func TestExportWithAuditJSON(t *testing.T) {
	root := repoRoot(t)
	cfg := filepath.Join(root, "testdata", "valid.yaml")
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.json")
	outPath := filepath.Join(dir, "export.csv")

	auditOut, _, code := runCLI(t, root, "audit", "--config", cfg, "--simulate", "--output", "json", "--threshold", "70")
	if code != 0 {
		t.Fatalf("audit exit %d", code)
	}
	if err := os.WriteFile(auditPath, []byte(strings.TrimSpace(auditOut)), 0o644); err != nil {
		t.Fatal(err)
	}

	csvOut, _, code := runCLI(t, root,
		"export", "--config", cfg,
		"--audit-json", auditPath,
		"--output", "file:"+outPath,
		"--simulate", "false",
	)
	if code != 0 {
		t.Fatalf("export exit %d: %s", code, csvOut)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	csv := string(data)
	if !strings.HasPrefix(csv, "timestamp;") {
		t.Fatalf("csv header: %q", csv[:min(40, len(csv))])
	}
	if !strings.Contains(csv, "electricity") {
		t.Fatal("missing meter type in export")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
