package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sysrqio/liegenpulse/internal/audit"
	"github.com/sysrqio/liegenpulse/internal/config"
	"github.com/sysrqio/liegenpulse/internal/export"
	"github.com/sysrqio/liegenpulse/internal/generate"
	"github.com/sysrqio/liegenpulse/internal/simulate"
)

var version = "0.1.0"

func main() {
	if err := newRoot().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "liegenpulse",
		Short: "EnEfG readiness audit for municipal meter configurations",
	}
	root.AddCommand(cmdAudit(), cmdGenerate(), cmdExport(), cmdVersion())
	return root
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("liegenpulse %s\n", version)
		},
	}
}

func cmdAudit() *cobra.Command {
	var configPath, output string
	var simulateFlag bool
	var threshold int
	var verbose bool

	c := &cobra.Command{
		Use:   "audit",
		Short: "Audit meters.yaml for EnEfG readiness",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.Load(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
				os.Exit(1)
			}
			issues := root.Validate()
			if hasBlockingIssues(issues) {
				for _, iss := range issues {
					fmt.Fprintf(os.Stderr, "validation: %s\n", formatIssue(iss))
				}
				os.Exit(1)
			}

			var reach simulate.Reachability
			if simulateFlag {
				reach, err = simulate.BuildReachability(root)
				if err != nil {
					return err
				}
			}

			result := audit.Run(root, reach, audit.Options{
				Simulate:  simulateFlag,
				Threshold: threshold,
			})

			if err := writeAuditOutput(result, output, verbose); err != nil {
				return err
			}

			if result.Score < float64(threshold) {
				os.Exit(2)
			}
			return nil
		},
	}
	c.Flags().StringVar(&configPath, "config", "meters.yaml", "Path to meters.yaml")
	c.Flags().StringVar(&output, "output", "stdout", "stdout|json|file:path")
	c.Flags().BoolVar(&simulateFlag, "simulate", true, "Use embedded reachability fixture (no live Modbus)")
	c.Flags().IntVar(&threshold, "threshold", 70, "Minimum passing score")
	c.Flags().BoolVar(&verbose, "verbose", false, "Verbose human output")
	return c
}

func hasBlockingIssues(issues []config.ValidationIssue) bool {
	for _, iss := range issues {
		if strings.Contains(iss.Message, "EnEfG readiness") {
			continue
		}
		return true
	}
	return false
}

func formatIssue(i config.ValidationIssue) string {
	if i.Building != "" && i.Meter != "" {
		return fmt.Sprintf("%s/%s: %s", i.Building, i.Meter, i.Message)
	}
	if i.Building != "" {
		return fmt.Sprintf("%s: %s", i.Building, i.Message)
	}
	return i.Message
}

func writeAuditOutput(result audit.Result, output string, verbose bool) error {
	var data []byte
	var err error
	switch {
	case output == "json":
		data, err = result.JSON()
	case strings.HasPrefix(output, "file:"):
		path := strings.TrimPrefix(output, "file:")
		var payload string
		if strings.HasSuffix(path, ".json") {
			data, err = result.JSON()
			if err != nil {
				return err
			}
			return os.WriteFile(path, data, 0o644)
		}
		payload = result.FormatHuman(verbose)
		return os.WriteFile(path, []byte(payload), 0o644)
	default:
		fmt.Print(result.FormatHuman(verbose))
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func cmdGenerate() *cobra.Command {
	var configPath, out string

	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate Telegraf TOML snippets from meters.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.Load(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
				os.Exit(1)
			}
			if issues := root.Validate(); len(issues) > 0 {
				for _, iss := range issues {
					if isSchemaError(iss) {
						fmt.Fprintf(os.Stderr, "validation: %s\n", formatIssue(iss))
						os.Exit(1)
					}
				}
			}
			toml := generate.Telegraf(root)
			if out == "" || out == "stdout" {
				fmt.Print(toml)
				return nil
			}
			if strings.HasPrefix(out, "file:") {
				return os.WriteFile(strings.TrimPrefix(out, "file:"), []byte(toml), 0o644)
			}
			return os.WriteFile(out, []byte(toml), 0o644)
		},
	}
	c.Flags().StringVar(&configPath, "config", "meters.yaml", "Path to meters.yaml")
	c.Flags().StringVar(&out, "output", "stdout", "stdout or file:path")
	return c
}

func isSchemaError(i config.ValidationIssue) bool {
	return strings.Contains(i.Message, "invalid") ||
		strings.Contains(i.Message, "duplicate") ||
		strings.Contains(i.Message, "required") && !strings.Contains(i.Message, "EnEfG")
}

func cmdExport() *cobra.Command {
	var configPath, auditJSON, out string
	var simulateFlag bool
	var threshold int

	c := &cobra.Command{
		Use:   "export",
		Short: "Export Kom.EMS-compatible CSV from config and optional audit",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := config.Load(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
				os.Exit(1)
			}

			var result *audit.Result
			if auditJSON != "" {
				data, err := os.ReadFile(auditJSON)
				if err != nil {
					return err
				}
				var r audit.Result
				if err := jsonUnmarshal(data, &r); err != nil {
					return err
				}
				result = &r
			} else if simulateFlag {
				reach, err := simulate.BuildReachability(root)
				if err != nil {
					return err
				}
				r := audit.Run(root, reach, audit.Options{Simulate: true, Threshold: threshold})
				result = &r
			}

			csv, err := export.KomEMS(root, result)
			if err != nil {
				return err
			}
			if out == "" || out == "stdout" {
				fmt.Print(csv)
				return nil
			}
			path := out
			if strings.HasPrefix(out, "file:") {
				path = strings.TrimPrefix(out, "file:")
			}
			return os.WriteFile(path, []byte(csv), 0o644)
		},
	}
	c.Flags().StringVar(&configPath, "config", "meters.yaml", "Path to meters.yaml")
	c.Flags().StringVar(&auditJSON, "audit-json", "", "Optional audit result JSON file")
	c.Flags().StringVar(&out, "output", "stdout", "stdout or file:path")
	c.Flags().BoolVar(&simulateFlag, "simulate", true, "Run simulate audit when no audit-json")
	c.Flags().IntVar(&threshold, "threshold", 70, "Threshold when simulating audit")
	return c
}

func jsonUnmarshal(data []byte, v *audit.Result) error {
	return json.Unmarshal(data, v)
}
